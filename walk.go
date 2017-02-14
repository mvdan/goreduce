// Heavily modivied version of Go's src/go/ast/walk.go

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import "go/ast"

func (r *reducer) walkIdentList(list []*ast.Ident, fn func(v interface{}) bool) {
	for _, x := range list {
		r.walkOther(x, fn)
	}
}

func (r *reducer) walkExprList(list []ast.Expr, fn func(v interface{}) bool) {
	for i := range list {
		r.walkExpr(&list[i], fn)
	}
}

func (r *reducer) walkStmtList(list *[]ast.Stmt, fn func(v interface{}) bool) {
	l := *list
	if len(l) == 0 || !fn(list) {
		return
	}
	for i := range l {
		r.walkStmt(&l[i], fn)
	}
}

func (r *reducer) walkDeclList(list []ast.Decl, fn func(v interface{}) bool) {
	for _, x := range list {
		r.walkOther(x, fn)
	}
}

func (r *reducer) walkStmt(stmt *ast.Stmt, fn func(v interface{}) bool) {
	r.stmt = stmt
	r.expr = nil
	r.walk(*stmt, fn)
}

func (r *reducer) walkExpr(expr *ast.Expr, fn func(v interface{}) bool) {
	r.stmt = nil
	r.expr = expr
	r.walk(*expr, fn)
}

func (r *reducer) walkOther(node ast.Node, fn func(v interface{}) bool) {
	r.stmt = nil
	r.expr = nil
	r.walk(node, fn)
}

func (r *reducer) walk(node ast.Node, fn func(v interface{}) bool) {
	if !fn(node) {
		return
	}
	switch x := node.(type) {
	// Fields
	case *ast.Field:
		r.walkIdentList(x.Names, fn)
		r.walkExpr(&x.Type, fn)
		if x.Tag != nil {
			r.walkOther(x.Tag, fn)
		}

	case *ast.FieldList:
		for _, f := range x.List {
			r.walkOther(f, fn)
		}

	// Expressions
	case *ast.BasicLit:

	case *ast.Ident:
		// nothing to do

	case *ast.Ellipsis:
		if x.Elt != nil {
			r.walkExpr(&x.Elt, fn)
		}

	case *ast.FuncLit:
		r.walkOther(x.Type, fn)
		r.walkOther(x.Body, fn)

	case *ast.CompositeLit:
		if x.Type != nil {
			r.walkExpr(&x.Type, fn)
		}
		r.walkExprList(x.Elts, fn)

	case *ast.ParenExpr:
		r.walkExpr(&x.X, fn)

	case *ast.SelectorExpr:
		r.walkExpr(&x.X, fn)
		r.walkOther(x.Sel, fn)

	case *ast.IndexExpr:
		r.walkExpr(&x.X, fn)
		r.walkExpr(&x.Index, fn)

	case *ast.SliceExpr:
		r.walkExpr(&x.X, fn)
		if x.Low != nil {
			r.walkExpr(&x.Low, fn)
		}
		if x.High != nil {
			r.walkExpr(&x.High, fn)
		}
		if x.Max != nil {
			r.walkExpr(&x.Max, fn)
		}

	case *ast.TypeAssertExpr:
		r.walkExpr(&x.X, fn)
		if x.Type != nil {
			r.walkExpr(&x.Type, fn)
		}

	case *ast.CallExpr:
		r.walkExpr(&x.Fun, fn)
		r.walkExprList(x.Args, fn)

	case *ast.StarExpr:
		r.walkExpr(&x.X, fn)

	case *ast.UnaryExpr:
		r.walkExpr(&x.X, fn)

	case *ast.BinaryExpr:
		r.walkExpr(&x.X, fn)
		r.walkExpr(&x.Y, fn)

	case *ast.KeyValueExpr:
		r.walkExpr(&x.Key, fn)
		r.walkExpr(&x.Value, fn)

	// Types
	case *ast.ArrayType:
		if x.Len != nil {
			r.walkExpr(&x.Len, fn)
		}
		r.walkExpr(&x.Elt, fn)

	case *ast.StructType:
		r.walkOther(x.Fields, fn)

	case *ast.FuncType:
		if x.Params != nil {
			r.walkOther(x.Params, fn)
		}
		if x.Results != nil {
			r.walkOther(x.Results, fn)
		}

	case *ast.InterfaceType:
		r.walkOther(x.Methods, fn)

	case *ast.MapType:
		r.walkExpr(&x.Key, fn)
		r.walkExpr(&x.Value, fn)

	case *ast.ChanType:
		r.walkExpr(&x.Value, fn)

	// Statements
	case *ast.DeclStmt:
		r.walkOther(x.Decl, fn)

	case *ast.LabeledStmt:
		r.walkOther(x.Label, fn)
		r.walkStmt(&x.Stmt, fn)

	case *ast.ExprStmt:
		r.walkExpr(&x.X, fn)

	case *ast.SendStmt:
		r.walkExpr(&x.Chan, fn)
		r.walkExpr(&x.Value, fn)

	case *ast.IncDecStmt:
		r.walkExpr(&x.X, fn)

	case *ast.AssignStmt:
		r.walkExprList(x.Lhs, fn)
		r.walkExprList(x.Rhs, fn)

	case *ast.GoStmt:
		r.walkOther(x.Call, fn)

	case *ast.DeferStmt:
		r.walkOther(x.Call, fn)

	case *ast.ReturnStmt:
		r.walkExprList(x.Results, fn)

	case *ast.BranchStmt:
		if x.Label != nil {
			r.walkOther(x.Label, fn)
		}

	case *ast.BlockStmt:
		r.walkStmtList(&x.List, fn)

	case *ast.IfStmt:
		if x.Init != nil {
			r.walkStmt(&x.Init, fn)
		}
		r.walkExpr(&x.Cond, fn)
		r.walkOther(x.Body, fn)
		if x.Else != nil {
			r.walkStmt(&x.Else, fn)
		}

	case *ast.CaseClause:
		r.walkExprList(x.List, fn)
		r.walkStmtList(&x.Body, fn)

	case *ast.SwitchStmt:
		if x.Init != nil {
			r.walkStmt(&x.Init, fn)
		}
		if x.Tag != nil {
			r.walkOther(x.Tag, fn)
		}
		r.walkOther(x.Body, fn)

	case *ast.TypeSwitchStmt:
		if x.Init != nil {
			r.walkStmt(&x.Init, fn)
		}
		r.walkStmt(&x.Assign, fn)
		r.walkOther(x.Body, fn)

	case *ast.CommClause:
		if x.Comm != nil {
			r.walkStmt(&x.Comm, fn)
		}
		r.walkStmtList(&x.Body, fn)

	case *ast.SelectStmt:
		r.walkOther(x.Body, fn)

	case *ast.ForStmt:
		if x.Init != nil {
			r.walkStmt(&x.Init, fn)
		}
		if x.Cond != nil {
			r.walkExpr(&x.Cond, fn)
		}
		if x.Post != nil {
			r.walkStmt(&x.Post, fn)
		}
		r.walkOther(x.Body, fn)

	case *ast.RangeStmt:
		if x.Key != nil {
			r.walkExpr(&x.Key, fn)
		}
		if x.Value != nil {
			r.walkExpr(&x.Value, fn)
		}
		r.walkExpr(&x.X, fn)
		r.walkOther(x.Body, fn)

	// Declarations
	case *ast.ImportSpec:
		if x.Name != nil {
			r.walkOther(x.Name, fn)
		}
		r.walkOther(x.Path, fn)

	case *ast.ValueSpec:
		r.walkIdentList(x.Names, fn)
		if x.Type != nil {
			r.walkExpr(&x.Type, fn)
		}
		r.walkExprList(x.Values, fn)

	case *ast.TypeSpec:
		r.walkOther(x.Name, fn)
		r.walkExpr(&x.Type, fn)

	case *ast.GenDecl:
		for _, s := range x.Specs {
			r.walkOther(s, fn)
		}

	case *ast.FuncDecl:
		if x.Recv != nil {
			r.walkOther(x.Recv, fn)
		}
		r.walkOther(x.Name, fn)
		r.walkOther(x.Type, fn)
		if x.Body != nil {
			r.walkOther(x.Body, fn)
		}

	// Files and packages
	case *ast.File:
		r.walkOther(x.Name, fn)
		r.walkDeclList(x.Decls, fn)

	case *ast.Package:
		for _, f := range x.Files {
			r.walkOther(f, fn)
		}
	}
}
