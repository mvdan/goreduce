// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import (
	"go/ast"
	"go/token"
)

// TODO: use x/tools/go/ssa?

// uses interface{} instead of ast.Node for node slices
func (r *reducer) reduceNode(v interface{}) {
	switch x := v.(type) {
	case *[]ast.Stmt:
		r.removeStmt(x)
	case *ast.IfStmt:
		switch {
		case r.changeStmt(x.Body):
			r.logChange(x, "if a { b } -> b")
		case x.Else != nil && r.changeStmt(x.Else):
			r.logChange(x, "if a { ... } else { c } -> c")
		}
	case *ast.BasicLit:
		r.reduceLit(x)
	case *ast.SliceExpr:
		r.reduceSlice(x)
	case *ast.BinaryExpr:
		switch {
		case r.changeExpr(x.X):
			r.logChange(x, "a %v b -> a", x.Op)
		case r.changeExpr(x.Y):
			r.logChange(x, "a %v b -> b", x.Op)
		}
	case *ast.ParenExpr:
		if r.changeExpr(x.X) {
			r.logChange(x, "(a) -> a")
		}
	case *ast.IndexExpr:
		if r.changeExpr(x.X) {
			r.logChange(x, "a[b] -> a")
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
}

func (r *reducer) removeStmt(list *[]ast.Stmt) {
	orig := *list
	if len(orig) == 0 {
		return
	}
	l := make([]ast.Stmt, len(orig)-1)
	for i, stmt := range orig {
		// discard those that will break compilation
		switch x := stmt.(type) {
		case *ast.DeclStmt, *ast.ReturnStmt:
			continue
		case *ast.AssignStmt:
			if x.Tok == token.DEFINE { // :=
				continue
			}
		}
		copy(l, orig[:i])
		copy(l[i:], orig[i+1:])
		*list = l
		if r.okChange() {
			r.logChange(stmt, "statement removed")
			return
		}
	}
	*list = orig
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
			r.logChange(l, `"foo" -> ""`)
		}
	case token.INT:
		if changeValue(`0`) {
			r.logChange(l, `123 -> 0`)
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
			r.logChange(orig, "a[b:] -> a[:]")
			return
		}
		if i == 0 {
			sl.Slice3 = true
		}
		*expr = orig
	}
}
