// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

// TODO: use x/tools/go/ssa?

// uses interface{} instead of ast.Node for node slices
func (r *reducer) reduceNode(v interface{}) bool {
	if r.didChange {
		return false
	}
	switch x := v.(type) {
	case *ast.File:
		origDecls := x.Decls
		for i, decl := range x.Decls {
			gd, _ := decl.(*ast.GenDecl)
			if gd == nil || len(gd.Specs) != 1 {
				// we (will) reduce multiple specs into
				// 1 in the *ast.GenDecl case
				continue
			}
			vs, _ := gd.Specs[0].(*ast.ValueSpec)
			if vs == nil {
				continue
			}
			for _, name := range vs.Names {
				if ast.IsExported(name.Name) {
					return true
				}
				if len(r.useIdents[r.info.Defs[name]]) > 0 {
					return true
				}
			}
			x.Decls = append(x.Decls[:i], x.Decls[i+1:]...)
			if r.okChange() {
				if gd.Tok == token.CONST {
					r.logChange(x, "removed const decl")
				} else {
					r.logChange(x, "removed var decl")
				}
				break
			}
			x.Decls = origDecls
		}
	case *ast.ImportSpec:
		name := ""
		if x.Name != nil {
			name = x.Name.Name
		}
		if name != "_" {
			return false
		}
		path, _ := strconv.Unquote(x.Path.Value)
		astutil.DeleteNamedImport(r.fset, r.file, name, path)
		if r.okChange() {
			r.logChange(x, "removed import")
		} else {
			astutil.AddNamedImport(r.fset, r.file, name, path)
		}
		return false
	case *[]ast.Stmt:
		r.removeStmt(x)
		r.inlineBlock(x)
	case *ast.IfStmt:
		undo := r.afterDelete(x.Init, x.Cond, x.Else)
		if r.changeStmt(x.Body) {
			r.logChange(x, "if a { b } -> { b }")
			break
		}
		undo()
		if x.Else != nil {
			undo := r.afterDelete(x.Init, x.Cond, x.Body)
			if r.changeStmt(x.Else) {
				r.logChange(x, "if a {...} else c -> c")
				break
			}
			undo()
		}
	case *ast.Ident:
		obj := r.info.Uses[x]
		if obj == nil { // declaration of ident, not its use
			break
		}
		if len(r.useIdents[obj]) > 1 { // used elsewhere
			break
		}
		if _, ok := obj.Type().(*types.Basic); !ok {
			break
		}
		declIdent := r.revDefs[obj]
		isVar := true
		var expr ast.Expr
		switch y := r.parents[declIdent].(type) {
		case *ast.ValueSpec:
			if gd := r.parents[y].(*ast.GenDecl); gd.Tok == token.CONST {
				isVar = false
			}
			for i, name := range y.Names {
				if name == declIdent {
					expr = y.Values[i]
					break
				}
			}
		case *ast.AssignStmt: // Tok must be := (DEFINE)
			for i, name := range y.Lhs {
				if name == declIdent {
					expr = y.Rhs[i]
					break
				}
			}
		}
		undo := r.afterDelete(x)
		if expr != nil && r.changeExpr(expr) {
			if isVar {
				r.logChange(x, "var inlined")
			} else {
				r.logChange(x, "const inlined")
			}
			break
		}
		undo()
	case *ast.BasicLit:
		r.reduceLit(x)
	case *ast.SliceExpr:
		r.reduceSlice(x)
	case *ast.CompositeLit:
		if len(x.Elts) == 0 {
			break
		}
		orig := x.Elts
		undo := r.afterDeleteExprs(x.Elts)
		if x.Elts = nil; r.okChange() {
			t := "T"
			switch x.Type.(type) {
			case *ast.ArrayType:
				t = "[]" + t
			}
			r.logChange(x, "%s{a, b} -> %s{}", t, t)
			break
		}
		undo()
		x.Elts = orig
	case *ast.BinaryExpr:
		undo := r.afterDelete(x.Y)
		if r.changeExpr(x.X) {
			r.logChange(x, "a %v b -> a", x.Op)
			break
		}
		undo()
		undo = r.afterDelete(x.X)
		if r.changeExpr(x.Y) {
			r.logChange(x, "a %v b -> b", x.Op)
			break
		}
		undo()
	case *ast.ParenExpr:
		if r.changeExpr(x.X) {
			r.logChange(x, "(a) -> a")
		}
	case *ast.IndexExpr:
		undo := r.afterDelete(x.Index)
		if r.changeExpr(x.X) {
			r.logChange(x, "a[b] -> a")
			break
		}
		undo()
	case *ast.StarExpr:
		if r.changeExpr(x.X) {
			r.logChange(x, "*a -> a")
		}
	case *ast.UnaryExpr:
		if r.changeExpr(x.X) {
			r.logChange(x, "%va -> a", x.Op)
		}
	case *ast.GoStmt:
		if r.changeStmt(&ast.ExprStmt{X: x.Call}) {
			r.logChange(x, "go a() -> a()")
		}
	case *ast.DeferStmt:
		if r.changeStmt(&ast.ExprStmt{X: x.Call}) {
			r.logChange(x, "defer a() -> a()")
		}
	}
	return true
}

func basicLitEqualsString(bl *ast.BasicLit, s string) bool {
	if bl.Kind != token.STRING {
		return false
	}
	unq, _ := strconv.Unquote(bl.Value)
	return unq == s
}

func (r *reducer) removeStmt(list *[]ast.Stmt) {
	orig := *list
	l := make([]ast.Stmt, len(orig)-1)
	seenTerminating := false
	for i, stmt := range orig {
		// discard those that will likely break compilation
		switch x := stmt.(type) {
		case *ast.DeclStmt:
			gd := x.Decl.(*ast.GenDecl)
			if !r.allUnusedNames(gd) {
				continue
			}
		case *ast.AssignStmt:
			if x.Tok == token.DEFINE { // :=
				continue
			}
		case *ast.ExprStmt:
			ce, _ := x.X.(*ast.CallExpr)
			if ce == nil {
				break
			}
			id, _ := ce.Fun.(*ast.Ident)
			if id != nil && id.Name == "panic" {
				seenTerminating = true
			}
		case *ast.ReturnStmt:
			if !seenTerminating {
				seenTerminating = true
				continue
			}
		}
		undo := r.afterDelete(stmt)
		copy(l, orig[:i])
		copy(l[i:], orig[i+1:])
		*list = l
		if r.okChange() {
			r.mergeLinesNode(stmt)
			r.logChange(stmt, "%s removed", nodeType(stmt))
			return
		}
		undo()
	}
	*list = orig
}

// allUnusedNames reports whether all delcs in a GenDecl are vars or
// consts with empty (underscore) or unused names.
func (r *reducer) allUnusedNames(gd *ast.GenDecl) bool {
	for _, spec := range gd.Specs {
		vs, _ := spec.(*ast.ValueSpec)
		if vs == nil {
			return false
		}
		for _, name := range vs.Names {
			if len(r.useIdents[r.info.Defs[name]]) > 0 {
				return false
			}
		}
	}
	return true
}

func nodeType(n ast.Node) string {
	s := fmt.Sprintf("%T", n)
	if i := strings.IndexByte(s, '.'); i >= 0 {
		s = s[i+1:]
	}
	return s
}

func (r *reducer) mergeLinesNode(node ast.Node) {
	r.mergeLines(node.Pos(), node.End()+1)
}

func (r *reducer) mergeLines(start, end token.Pos) {
	file := r.fset.File(start)
	l1 := file.Line(start)
	l2 := file.Line(end)
	for l1 < l2 {
		file.MergeLine(l1)
		l1++
	}
}

func (r *reducer) inlineBlock(list *[]ast.Stmt) {
	orig := *list
	type undoIdent struct {
		id   *ast.Ident
		name string
	}
	var undoIdents []undoIdent
	for i, stmt := range orig {
		bl, _ := stmt.(*ast.BlockStmt)
		if bl == nil {
			continue
		}
		fixScopeNames := func(node ast.Node) bool {
			switch x := node.(type) {
			case *ast.BlockStmt:
				return false
			case *ast.Ident:
				obj := r.info.Defs[x]
				if obj == nil { // use, not decl
					break
				}
				scope := obj.Parent()
				if scope.Parent().Lookup(x.Name) == nil {
					break
				}
				origName := x.Name
				newName := x.Name + "_"
				for scope.Lookup(newName) != nil {
					newName += "_"
				}
				x.Name = newName
				for _, use := range r.useIdents[obj] {
					undoIdents = append(undoIdents, undoIdent{
						id:   use,
						name: origName,
					})
					use.Name = newName
				}
			}
			return true
		}
		for _, stmt := range bl.List {
			ast.Inspect(stmt, fixScopeNames)
		}
		var l []ast.Stmt
		l = append(l, orig[:i]...)
		l = append(l, bl.List...)
		l = append(l, orig[i+1:]...)
		*list = l
		if r.okChange() {
			r.mergeLines(bl.Pos(), bl.List[0].Pos())
			r.mergeLines(bl.List[len(bl.List)-1].End(), bl.End())
			r.logChange(stmt, "block inlined")
			return
		}
	}
	for _, ui := range undoIdents {
		ui.id.Name = ui.name
	}
	*list = orig
}

func (r *reducer) afterDeleteExprs(exprs []ast.Expr) (undo func()) {
	nodes := make([]ast.Node, len(exprs))
	for i, expr := range exprs {
		nodes[i] = expr
	}
	return r.afterDelete(nodes...)
}

func (r *reducer) afterDelete(nodes ...ast.Node) (undo func()) {
	type redoImp struct {
		imp  *ast.ImportSpec
		name *ast.Ident
	}
	var imps []redoImp
	type redoVar struct {
		id   *ast.Ident
		name string
		asgn *ast.AssignStmt
	}
	var vars []redoVar

	for _, obj := range r.unusedAfterDelete(nodes...) {
		switch x := obj.(type) {
		case *types.PkgName:
			name := x.Name()
			if x.Imported().Name() == name {
				// import wasn't named
				name = ""
			}
			path := x.Imported().Path()
			ast.Inspect(r.file, func(node ast.Node) bool {
				imp, _ := node.(*ast.ImportSpec)
				if imp == nil {
					return true
				}
				if imp.Name != nil && imp.Name.Name != name {
					return true
				}
				if !basicLitEqualsString(imp.Path, path) {
					return true
				}
				imps = append(imps, redoImp{
					imp:  imp,
					name: imp.Name,
				})
				imp.Name = &ast.Ident{
					NamePos: imp.Path.Pos(),
					Name:    "_",
				}
				return true
			})
		case *types.Var:
			ast.Inspect(r.file, func(node ast.Node) bool {
				switch x := node.(type) {
				case *ast.Ident:
					vars = append(vars, redoVar{x, x.Name, nil})
					if r.info.Defs[x] == obj {
						x.Name = "_"
					}
				case *ast.AssignStmt:
					// TODO: support more complex assigns
					if len(x.Lhs) != 1 {
						return false
					}
					id, _ := x.Lhs[0].(*ast.Ident)
					if id == nil || r.info.Defs[id] != obj {
						return false
					}
					vars = append(vars, redoVar{id, id.Name, x})
					id.Name, x.Tok = "_", token.ASSIGN
					return false
				}
				return true
			})
		}
	}
	return func() {
		for _, imp := range imps {
			// go/types doesn't treat an empty name
			// literal the same way as no literal
			imp.imp.Name = imp.name
		}
		for _, rvar := range vars {
			rvar.id.Name = rvar.name
			if rvar.asgn != nil {
				rvar.asgn.Tok = token.DEFINE
			}
		}
	}
}

func (r *reducer) unusedAfterDelete(nodes ...ast.Node) (objs []types.Object) {
	remaining := make(map[types.Object]int)
	for _, node := range nodes {
		if node == nil {
			continue // for convenience
		}
		ast.Inspect(node, func(node ast.Node) bool {
			id, _ := node.(*ast.Ident)
			obj := r.info.Uses[id]
			if id == nil || obj == nil {
				return true
			}
			if num, e := remaining[obj]; e {
				if num == 1 {
					objs = append(objs, obj)
				}
				remaining[obj]--
			} else if ids, e := r.useIdents[obj]; e {
				if len(ids) == 1 {
					objs = append(objs, obj)
				} else {
					remaining[obj] = len(ids) - 1
				}
			}
			return true
		})
	}
	return
}

func (r *reducer) changeStmt(stmt ast.Stmt) bool {
	orig := *r.stmt
	if *r.stmt = stmt; r.okChange() {
		return true
	}
	*r.stmt = orig
	return false
}

func (r *reducer) changeExpr(expr ast.Expr) bool {
	orig := *r.expr
	if *r.expr = expr; r.okChange() {
		return true
	}
	*r.expr = orig
	return false
}

func (r *reducer) reduceLit(l *ast.BasicLit) {
	orig := l.Value
	changeValue := func(val string) bool {
		if l.Value == val {
			return false
		}
		if l.Value = val; r.okChange() {
			return true
		}
		l.Value = orig
		return false
	}
	switch l.Kind {
	case token.STRING:
		if changeValue(`""`) {
			if len(orig) > 10 {
				orig = fmt.Sprintf(`%s..."`, orig[:7])
			}
			r.logChange(l, `%s -> ""`, orig)
		}
	case token.INT:
		if changeValue(`0`) {
			if len(orig) > 10 {
				orig = fmt.Sprintf(`%s...`, orig[:7])
			}
			r.logChange(l, `%s -> 0`, orig)
		}
	}
}

func (r *reducer) reduceSlice(sl *ast.SliceExpr) {
	if r.changeExpr(sl.X) {
		r.logChange(sl, "a[b:] -> a")
		return
	}
	for i, expr := range [...]*ast.Expr{
		&sl.Max,
		&sl.High,
		&sl.Low,
	} {
		orig := *expr
		if orig == nil {
			continue
		}
		if i == 0 {
			sl.Slice3 = false
		}
		if *expr = nil; r.okChange() {
			switch i {
			case 0:
				r.logChange(orig, "a[b:c:d] -> a[b:c]")
			case 1:
				r.logChange(orig, "a[b:c] -> a[b:]")
			case 2:
				r.logChange(orig, "a[b:c] -> a[:c]")
			}
			return
		}
		if i == 0 {
			sl.Slice3 = true
		}
		*expr = orig
	}
}
