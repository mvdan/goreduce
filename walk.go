// Heavily modivied version of Go's src/go/ast/walk.go

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import "go/ast"

type walker struct {
	stmt *ast.Stmt
	expr *ast.Expr
}

func (w *walker) walkIdentList(list []*ast.Ident, fn func(v interface{}) bool) {
	for _, x := range list {
		w.walkOther(x, fn)
	}
}

func (w *walker) walkExprList(list []ast.Expr, fn func(v interface{}) bool) {
	for i := range list {
		w.walkExpr(&list[i], fn)
	}
}

func (w *walker) walkStmtList(list *[]ast.Stmt, fn func(v interface{}) bool) {
	l := *list
	if len(l) == 0 || !fn(list) {
		return
	}
	for i := range l {
		w.walkStmt(&l[i], fn)
	}
}

func (w *walker) walkDeclList(list []ast.Decl, fn func(v interface{}) bool) {
	for _, x := range list {
		w.walkOther(x, fn)
	}
}

func (w *walker) walkStmt(stmt *ast.Stmt, fn func(v interface{}) bool) {
	w.stmt = stmt
	w.expr = nil
	w.walk(*stmt, fn)
}

func (w *walker) walkExpr(expr *ast.Expr, fn func(v interface{}) bool) {
	w.stmt = nil
	w.expr = expr
	w.walk(*expr, fn)
}

func (w *walker) walkOther(node ast.Node, fn func(v interface{}) bool) {
	w.stmt = nil
	w.expr = nil
	w.walk(node, fn)
}

func (w *walker) walk(node ast.Node, fn func(v interface{}) bool) {
	if !fn(node) {
		return
	}
	switch x := node.(type) {
	// Fields
	case *ast.Field:
		w.walkIdentList(x.Names, fn)
		w.walkExpr(&x.Type, fn)
		if x.Tag != nil {
			w.walkOther(x.Tag, fn)
		}

	case *ast.FieldList:
		for _, f := range x.List {
			w.walkOther(f, fn)
		}

	// Expressions
	case *ast.BasicLit:

	case *ast.Ident:
		// nothing to do

	case *ast.Ellipsis:
		if x.Elt != nil {
			w.walkExpr(&x.Elt, fn)
		}

	case *ast.FuncLit:
		w.walkOther(x.Type, fn)
		w.walkOther(x.Body, fn)

	case *ast.CompositeLit:
		if x.Type != nil {
			w.walkExpr(&x.Type, fn)
		}
		w.walkExprList(x.Elts, fn)

	case *ast.ParenExpr:
		w.walkExpr(&x.X, fn)

	case *ast.SelectorExpr:
		w.walkExpr(&x.X, fn)
		w.walkOther(x.Sel, fn)

	case *ast.IndexExpr:
		w.walkExpr(&x.X, fn)
		w.walkExpr(&x.Index, fn)

	case *ast.SliceExpr:
		w.walkExpr(&x.X, fn)
		if x.Low != nil {
			w.walkExpr(&x.Low, fn)
		}
		if x.High != nil {
			w.walkExpr(&x.High, fn)
		}
		if x.Max != nil {
			w.walkExpr(&x.Max, fn)
		}

	case *ast.TypeAssertExpr:
		w.walkExpr(&x.X, fn)
		if x.Type != nil {
			w.walkExpr(&x.Type, fn)
		}

	case *ast.CallExpr:
		w.walkExpr(&x.Fun, fn)
		w.walkExprList(x.Args, fn)

	case *ast.StarExpr:
		w.walkExpr(&x.X, fn)

	case *ast.UnaryExpr:
		w.walkExpr(&x.X, fn)

	case *ast.BinaryExpr:
		w.walkExpr(&x.X, fn)
		w.walkExpr(&x.Y, fn)

	case *ast.KeyValueExpr:
		w.walkExpr(&x.Key, fn)
		w.walkExpr(&x.Value, fn)

	// Types
	case *ast.ArrayType:
		if x.Len != nil {
			w.walkExpr(&x.Len, fn)
		}
		w.walkExpr(&x.Elt, fn)

	case *ast.StructType:
		w.walkOther(x.Fields, fn)

	case *ast.FuncType:
		if x.Params != nil {
			w.walkOther(x.Params, fn)
		}
		if x.Results != nil {
			w.walkOther(x.Results, fn)
		}

	case *ast.InterfaceType:
		w.walkOther(x.Methods, fn)

	case *ast.MapType:
		w.walkExpr(&x.Key, fn)
		w.walkExpr(&x.Value, fn)

	case *ast.ChanType:
		w.walkExpr(&x.Value, fn)

	// Statements
	case *ast.DeclStmt:
		w.walkOther(x.Decl, fn)

	case *ast.LabeledStmt:
		w.walkOther(x.Label, fn)
		w.walkStmt(&x.Stmt, fn)

	case *ast.ExprStmt:
		w.walkExpr(&x.X, fn)

	case *ast.SendStmt:
		w.walkExpr(&x.Chan, fn)
		w.walkExpr(&x.Value, fn)

	case *ast.IncDecStmt:
		w.walkExpr(&x.X, fn)

	case *ast.AssignStmt:
		w.walkExprList(x.Lhs, fn)
		w.walkExprList(x.Rhs, fn)

	case *ast.GoStmt:
		w.walkOther(x.Call, fn)

	case *ast.DeferStmt:
		w.walkOther(x.Call, fn)

	case *ast.ReturnStmt:
		w.walkExprList(x.Results, fn)

	case *ast.BranchStmt:
		if x.Label != nil {
			w.walkOther(x.Label, fn)
		}

	case *ast.BlockStmt:
		w.walkStmtList(&x.List, fn)

	case *ast.IfStmt:
		if x.Init != nil {
			w.walkStmt(&x.Init, fn)
		}
		w.walkExpr(&x.Cond, fn)
		w.walkOther(x.Body, fn)
		if x.Else != nil {
			w.walkStmt(&x.Else, fn)
		}

	case *ast.CaseClause:
		w.walkExprList(x.List, fn)
		w.walkStmtList(&x.Body, fn)

	case *ast.SwitchStmt:
		if x.Init != nil {
			w.walkStmt(&x.Init, fn)
		}
		if x.Tag != nil {
			w.walkOther(x.Tag, fn)
		}
		w.walkOther(x.Body, fn)

	case *ast.TypeSwitchStmt:
		if x.Init != nil {
			w.walkStmt(&x.Init, fn)
		}
		w.walkStmt(&x.Assign, fn)
		w.walkOther(x.Body, fn)

	case *ast.CommClause:
		if x.Comm != nil {
			w.walkStmt(&x.Comm, fn)
		}
		w.walkStmtList(&x.Body, fn)

	case *ast.SelectStmt:
		w.walkOther(x.Body, fn)

	case *ast.ForStmt:
		if x.Init != nil {
			w.walkStmt(&x.Init, fn)
		}
		if x.Cond != nil {
			w.walkExpr(&x.Cond, fn)
		}
		if x.Post != nil {
			w.walkStmt(&x.Post, fn)
		}
		w.walkOther(x.Body, fn)

	case *ast.RangeStmt:
		if x.Key != nil {
			w.walkExpr(&x.Key, fn)
		}
		if x.Value != nil {
			w.walkExpr(&x.Value, fn)
		}
		w.walkExpr(&x.X, fn)
		w.walkOther(x.Body, fn)

	// Declarations
	case *ast.ImportSpec:
		if x.Name != nil {
			w.walkOther(x.Name, fn)
		}
		w.walkOther(x.Path, fn)

	case *ast.ValueSpec:
		w.walkIdentList(x.Names, fn)
		if x.Type != nil {
			w.walkExpr(&x.Type, fn)
		}
		w.walkExprList(x.Values, fn)

	case *ast.TypeSpec:
		w.walkOther(x.Name, fn)
		w.walkExpr(&x.Type, fn)

	case *ast.GenDecl:
		for _, s := range x.Specs {
			w.walkOther(s, fn)
		}

	case *ast.FuncDecl:
		if x.Recv != nil {
			w.walkOther(x.Recv, fn)
		}
		w.walkOther(x.Name, fn)
		w.walkOther(x.Type, fn)
		if x.Body != nil {
			w.walkOther(x.Body, fn)
		}

	// Files and packages
	case *ast.File:
		w.walkOther(x.Name, fn)
		w.walkDeclList(x.Decls, fn)

	case *ast.Package:
		for _, f := range x.Files {
			w.walkOther(f, fn)
		}
	}
}
