// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"
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
		if x.Name == nil || x.Name.Name != "_" { // used
			return false
		}
		undo := r.removeSpec(x)
		if r.okChange() {
			r.logChange(x, "removed import")
		} else {
			undo()
		}
		return false
	case *[]ast.Stmt:
		if len(*x) == 1 { // we already tried removing the parent
			break
		}
		r.removeStmt(x)
	case *ast.BlockStmt:
		if r.canReplaceStmts(x) {
			r.inlineBlock(x)
		}
	case *ast.IfStmt:
		if len(x.Body.List) > 0 {
			r.afterDelete(x.Init, x.Cond, x.Else)
			if r.changeStmt(x.Body) {
				r.logChange(x, "if a { b } -> b")
				break
			}
		}
		if x.Else != nil {
			bl, _ := x.Else.(*ast.BlockStmt)
			if bl != nil && len(bl.List) < 1 {
				break
			}
			r.afterDelete(x.Init, x.Cond, x.Body)
			if r.changeStmt(x.Else) {
				r.logChange(x, "if a {...} else c -> c")
				break
			}
		}
	case *ast.SwitchStmt:
		if x.Init != nil || len(x.Body.List) != 1 {
			break
		}
		cs := x.Body.List[0].(*ast.CaseClause)
		if r.replaceStmts(x, cs.Body) {
			r.logChange(cs, "case inlined")
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
			isVar = r.parents[y].(*ast.GenDecl).Tok == token.VAR
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
		if expr == nil {
			break
		}
		r.afterDelete(x)
		if r.changeExpr(expr) {
			if isVar {
				r.logChange(x, "var inlined")
			} else {
				r.logChange(x, "const inlined")
			}
			break
		}
	case *ast.BasicLit:
		r.reduceLit(x)
	case *ast.SliceExpr:
		r.reduceSlice(x)
	case *ast.CompositeLit:
		if len(x.Elts) == 0 {
			break
		}
		orig := x.Elts
		r.afterDeleteExprs(x.Elts)
		if x.Elts = nil; r.okChange() {
			t := "T"
			switch x.Type.(type) {
			case *ast.ArrayType:
				t = "[]" + t
			}
			r.logChange(x, "%s{a, b} -> %s{}", t, t)
			break
		}
		x.Elts = orig
	case *ast.BinaryExpr:
		r.afterDelete(x.Y)
		if r.changeExpr(x.X) {
			r.logChange(x, "a %v b -> a", x.Op)
			break
		}
		r.afterDelete(x.X)
		if r.changeExpr(x.Y) {
			r.logChange(x, "a %v b -> b", x.Op)
			break
		}
	case *ast.ParenExpr:
		if r.changeExpr(x.X) {
			r.logChange(x, "(a) -> a")
		}
	case *ast.IndexExpr:
		r.afterDelete(x.Index)
		if r.changeExpr(x.X) {
			r.logChange(x, "a[b] -> a")
			break
		}
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

func (r *reducer) removeSpec(spec ast.Spec) (undo func()) {
	gd := r.parents[spec].(*ast.GenDecl)
	oldSpecs := gd.Specs
	for i, sp := range oldSpecs {
		if sp == spec {
			gd.Specs = append(gd.Specs[:i], gd.Specs[i+1:]...)
			break
		}
	}
	f := r.parents[gd].(*ast.File)
	oldDecls := f.Decls
	if len(gd.Specs) == 0 { // remove decl too
		for i, decl := range oldDecls {
			if decl == gd {
				f.Decls = append(f.Decls[:i], f.Decls[i+1:]...)
				break
			}
		}
	}
	return func() {
		gd.Specs = oldSpecs
		f.Decls = oldDecls
	}
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
			if id != nil && id.Name == "panic" && !seenTerminating {
				seenTerminating = true
				continue
			}
		case *ast.ReturnStmt:
			if !seenTerminating {
				seenTerminating = true
				continue
			}
		}
		r.afterDelete(stmt)
		copy(l, orig[:i])
		copy(l[i:], orig[i+1:])
		*list = l
		if r.okChange() {
			if i+i < len(orig) {
				r.mergeLines(stmt.Pos(), orig[i+1].End())
			} else {
				r.mergeLines(stmt.Pos(), stmt.End()+1)
			}
			r.logChange(stmt, "%s removed", nodeType(stmt))
			return
		}
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

func (r *reducer) mergeLines(start, end token.Pos) {
	file := r.fset.File(start)
	l1 := file.Line(start)
	l2 := file.Line(end)
	for l1 < l2 && l1 < file.LineCount() {
		file.MergeLine(l1)
		l1++
	}
}

func setPos(node ast.Node, pos token.Pos) {
	switch x := node.(type) {
	case *ast.BasicLit:
		x.ValuePos = pos
	case *ast.Ident:
		x.NamePos = pos
	case *ast.StarExpr:
		x.Star = pos
	case *ast.IndexExpr:
		setPos(x.X, pos)
	case *ast.ExprStmt:
		setPos(x.X, pos)
	case *ast.CompositeLit:
		if x.Type != nil {
			setPos(x.Type, pos)
		} else {
			x.Lbrace = pos
		}
	case *ast.CallExpr:
		setPos(x.Fun, pos)
	case *ast.ArrayType:
		x.Lbrack = pos
	}
}

func (r *reducer) adaptBlockNames(bl *ast.BlockStmt) (undo func()) {
	type undoIdent struct {
		id   *ast.Ident
		name string
	}
	var undoIdents []undoIdent
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
			newName := x.Name
			for scope.Lookup(newName) != nil {
				newName += "_"
			}
			for _, use := range r.useIdents[obj] {
				undoIdents = append(undoIdents, undoIdent{
					id:   use,
					name: x.Name,
				})
				use.Name = newName
			}
			x.Name = newName
		}
		return true
	}
	for _, stmt := range bl.List {
		ast.Inspect(stmt, fixScopeNames)
	}
	return func() {
		for _, ui := range undoIdents {
			ui.id.Name = ui.name
		}
	}
}

func (r *reducer) inlineBlock(bl *ast.BlockStmt) {
	undo := r.adaptBlockNames(bl)
	if r.replaceStmts(bl, bl.List) {
		r.logChange(bl, "block inlined")
		return
	}
	undo()
}

func (r *reducer) afterDeleteExprs(exprs []ast.Expr) {
	nodes := make([]ast.Node, len(exprs))
	for i, expr := range exprs {
		nodes[i] = expr
	}
	r.afterDelete(nodes...)
}

func (r *reducer) afterDelete(nodes ...ast.Node) {
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
			for _, imp := range r.file.Imports {
				if imp.Name != nil && imp.Name.Name != name {
					continue
				}
				unq, _ := strconv.Unquote(imp.Path.Value)
				if unq != path {
					continue
				}
				imps = append(imps, redoImp{
					imp:  imp,
					name: imp.Name,
				})
				imp.Name = &ast.Ident{
					NamePos: imp.Path.Pos(),
					Name:    "_",
				}
			}
		case *types.Var:
			declIdent := r.revDefs[x]
			vars = append(vars, redoVar{declIdent, declIdent.Name, nil})
			switch y := r.parents[declIdent].(type) {
			case *ast.AssignStmt: // Tok must be := (DEFINE)
				vars = append(vars, redoVar{declIdent, declIdent.Name, y})
				if len(y.Lhs) == 1 { // TODO: this is sloppy
					y.Tok = token.ASSIGN
				}
			}
			declIdent.Name = "_"
		}
	}
	if len(imps)+len(vars) == 0 {
		return
	}
	r.deleteKeepUnchanged = func() {
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
	if bl, _ := stmt.(*ast.BlockStmt); bl != nil {
		undo := r.adaptBlockNames(bl)
		if r.replaceStmts(*r.stmt, bl.List) {
			return true
		}
		undo()
	}
	orig := *r.stmt
	if *r.stmt = stmt; r.okChange() {
		setPos(stmt, orig.Pos())
		r.parents[stmt] = r.parents[orig]
		return true
	}
	*r.stmt = orig
	return false
}

func (r *reducer) changeExpr(expr ast.Expr) bool {
	orig := *r.expr
	if *r.expr = expr; r.okChange() {
		setPos(expr, orig.Pos())
		r.mergeLines(orig.Pos(), expr.Pos())
		r.mergeLines(expr.End(), orig.End())
		r.parents[expr] = r.parents[orig]
		return true
	}
	*r.expr = orig
	return false
}

func (r *reducer) canReplaceStmts(old ast.Stmt) bool {
	switch r.parents[old].(type) {
	case *ast.BlockStmt:
		return true
	default: // e.g. a func body, cannot inline
		return false
	}
}

func (r *reducer) replaceStmts(old ast.Stmt, with []ast.Stmt) bool {
	var stmts *[]ast.Stmt
	switch x := r.parents[old].(type) {
	case *ast.BlockStmt:
		stmts = &x.List
	default: // e.g. a func body, cannot inline
		return false
	}
	orig := *stmts
	i := 0
	for ; i < len(orig); i++ {
		if orig[i] == old {
			break
		}
	}
	l := make([]ast.Stmt, 0, (len(orig)+len(with))-1)
	l = append(l, orig[:i]...)
	l = append(l, with...)
	l = append(l, orig[i+1:]...)
	*stmts = l
	if r.okChange() {
		r.mergeLines(old.Pos(), with[0].Pos())
		r.mergeLines(with[len(with)-1].End(), old.End())
		setPos(with[0], old.Pos())
		for _, stmt := range with {
			r.parents[stmt] = r.parents[old]
		}
		return true
	}
	*stmts = orig
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
	r.afterDelete(sl.Low, sl.High, sl.Max)
	if r.changeExpr(sl.X) {
		r.logChange(sl, "a[b:] -> a")
		return
	}
	show := func(sl *ast.SliceExpr) string {
		buf := bytes.NewBufferString("a[")
		if sl.Low != nil {
			buf.WriteByte('l')
		}
		buf.WriteByte(':')
		if sl.High != nil {
			buf.WriteByte('h')
		}
		if sl.Slice3 {
			buf.WriteByte(':')
			buf.WriteByte('m')
		}
		buf.WriteByte(']')
		return buf.String()
	}
	origShow := show(sl)
	for i, expr := range [...]*ast.Expr{&sl.Max, &sl.High, &sl.Low} {
		orig := *expr
		if orig == nil {
			continue
		}
		if i == 0 {
			sl.Slice3 = false
		}
		r.afterDelete(orig)
		if *expr = nil; r.okChange() {
			r.logChange(orig, "%s -> %s", origShow, show(sl))
			return
		}
		if i == 0 {
			sl.Slice3 = true
		}
		*expr = orig
	}
}
