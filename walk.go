// Heavily modivied version of Go's src/go/ast/walk.go

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import "go/ast"

type walkItem struct {
	v    interface{}
	stmt *ast.Stmt
	expr *ast.Expr
}

type walker struct {
	fn    func(v interface{}) bool
	queue []walkItem
	stmt  *ast.Stmt
	expr  *ast.Expr
}

func (w *walker) walk(v interface{}, fn func(interface{}) bool) {
	w.queue = w.queue[:0]
	w.fn = fn
	w.walkOther(v)
	for len(w.queue) > 0 {
		wi := w.queue[0]
		w.queue = w.queue[1:]
		w.stmt, w.expr = wi.stmt, wi.expr
		w.walkSingle(wi.v)
	}
}

func (w *walker) walkIdentList(list []*ast.Ident) {
	for _, x := range list {
		w.walkOther(x)
	}
}

func (w *walker) walkExprList(list []ast.Expr) {
	for i := range list {
		w.walkExpr(&list[i])
	}
}

func (w *walker) walkStmtList(list *[]ast.Stmt) {
	if len(*list) > 0 {
		w.queue = append(w.queue, walkItem{v: list})
	}
}

func (w *walker) walkDeclList(list []ast.Decl) {
	for _, x := range list {
		w.walkOther(x)
	}
}

func (w *walker) walkStmt(stmt *ast.Stmt) {
	w.queue = append(w.queue, walkItem{v: *stmt, stmt: stmt})
}

func (w *walker) walkExpr(expr *ast.Expr) {
	w.queue = append(w.queue, walkItem{v: *expr, expr: expr})
}

func (w *walker) walkOther(v interface{}) {
	w.queue = append(w.queue, walkItem{v: v})
}

func (w *walker) walkSingle(v interface{}) {
	if !w.fn(v) {
		return
	}
	switch x := v.(type) {
	// Node lists
	case *[]ast.Stmt:
		l := *x
		for i := range l {
			w.walkStmt(&l[i])
		}
	// Fields
	case *ast.Field:
		w.walkIdentList(x.Names)
		w.walkExpr(&x.Type)
		if x.Tag != nil {
			w.walkOther(x.Tag)
		}

	case *ast.FieldList:
		for _, f := range x.List {
			w.walkOther(f)
		}

	// Expressions
	case *ast.BasicLit:

	case *ast.Ident:
		// nothing to do

	case *ast.Ellipsis:
		if x.Elt != nil {
			w.walkExpr(&x.Elt)
		}

	case *ast.FuncLit:
		w.walkOther(x.Type)
		w.walkOther(x.Body)

	case *ast.CompositeLit:
		if x.Type != nil {
			w.walkExpr(&x.Type)
		}
		w.walkExprList(x.Elts)

	case *ast.ParenExpr:
		w.walkExpr(&x.X)

	case *ast.SelectorExpr:
		w.walkExpr(&x.X)
		w.walkOther(x.Sel)

	case *ast.IndexExpr:
		w.walkExpr(&x.X)
		w.walkExpr(&x.Index)

	case *ast.SliceExpr:
		w.walkExpr(&x.X)
		if x.Low != nil {
			w.walkExpr(&x.Low)
		}
		if x.High != nil {
			w.walkExpr(&x.High)
		}
		if x.Max != nil {
			w.walkExpr(&x.Max)
		}

	case *ast.TypeAssertExpr:
		w.walkExpr(&x.X)
		if x.Type != nil {
			w.walkExpr(&x.Type)
		}

	case *ast.CallExpr:
		w.walkExpr(&x.Fun)
		w.walkExprList(x.Args)

	case *ast.StarExpr:
		w.walkExpr(&x.X)

	case *ast.UnaryExpr:
		w.walkExpr(&x.X)

	case *ast.BinaryExpr:
		w.walkExpr(&x.X)
		w.walkExpr(&x.Y)

	case *ast.KeyValueExpr:
		w.walkExpr(&x.Key)
		w.walkExpr(&x.Value)

	// Types
	case *ast.ArrayType:
		if x.Len != nil {
			w.walkExpr(&x.Len)
		}
		w.walkExpr(&x.Elt)

	case *ast.StructType:
		w.walkOther(x.Fields)

	case *ast.FuncType:
		if x.Params != nil {
			w.walkOther(x.Params)
		}
		if x.Results != nil {
			w.walkOther(x.Results)
		}

	case *ast.InterfaceType:
		w.walkOther(x.Methods)

	case *ast.MapType:
		w.walkExpr(&x.Key)
		w.walkExpr(&x.Value)

	case *ast.ChanType:
		w.walkExpr(&x.Value)

	// Statements
	case *ast.DeclStmt:
		w.walkOther(x.Decl)

	case *ast.LabeledStmt:
		w.walkOther(x.Label)
		w.walkStmt(&x.Stmt)

	case *ast.ExprStmt:
		w.walkExpr(&x.X)

	case *ast.SendStmt:
		w.walkExpr(&x.Chan)
		w.walkExpr(&x.Value)

	case *ast.IncDecStmt:
		w.walkExpr(&x.X)

	case *ast.AssignStmt:
		w.walkExprList(x.Lhs)
		w.walkExprList(x.Rhs)

	case *ast.GoStmt:
		w.walkOther(x.Call)

	case *ast.DeferStmt:
		w.walkOther(x.Call)

	case *ast.ReturnStmt:
		w.walkExprList(x.Results)

	case *ast.BranchStmt:
		if x.Label != nil {
			w.walkOther(x.Label)
		}

	case *ast.BlockStmt:
		w.walkStmtList(&x.List)

	case *ast.IfStmt:
		if x.Init != nil {
			w.walkStmt(&x.Init)
		}
		w.walkExpr(&x.Cond)
		w.walkOther(x.Body)
		if x.Else != nil {
			w.walkStmt(&x.Else)
		}

	case *ast.CaseClause:
		w.walkExprList(x.List)
		w.walkStmtList(&x.Body)

	case *ast.SwitchStmt:
		if x.Init != nil {
			w.walkStmt(&x.Init)
		}
		if x.Tag != nil {
			w.walkOther(x.Tag)
		}
		w.walkOther(x.Body)

	case *ast.TypeSwitchStmt:
		if x.Init != nil {
			w.walkStmt(&x.Init)
		}
		w.walkStmt(&x.Assign)
		w.walkOther(x.Body)

	case *ast.CommClause:
		if x.Comm != nil {
			w.walkStmt(&x.Comm)
		}
		w.walkStmtList(&x.Body)

	case *ast.SelectStmt:
		w.walkOther(x.Body)

	case *ast.ForStmt:
		if x.Init != nil {
			w.walkStmt(&x.Init)
		}
		if x.Cond != nil {
			w.walkExpr(&x.Cond)
		}
		if x.Post != nil {
			w.walkStmt(&x.Post)
		}
		w.walkOther(x.Body)

	case *ast.RangeStmt:
		if x.Key != nil {
			w.walkExpr(&x.Key)
		}
		if x.Value != nil {
			w.walkExpr(&x.Value)
		}
		w.walkExpr(&x.X)
		w.walkOther(x.Body)

	// Declarations
	case *ast.ImportSpec:
		if x.Name != nil {
			w.walkOther(x.Name)
		}
		w.walkOther(x.Path)

	case *ast.ValueSpec:
		w.walkIdentList(x.Names)
		if x.Type != nil {
			w.walkExpr(&x.Type)
		}
		w.walkExprList(x.Values)

	case *ast.TypeSpec:
		w.walkOther(x.Name)
		w.walkExpr(&x.Type)

	case *ast.GenDecl:
		for _, s := range x.Specs {
			w.walkOther(s)
		}

	case *ast.FuncDecl:
		if x.Recv != nil {
			w.walkOther(x.Recv)
		}
		w.walkOther(x.Name)
		w.walkOther(x.Type)
		if x.Body != nil {
			w.walkOther(x.Body)
		}

	// Files and packages
	case *ast.File:
		w.walkOther(x.Name)
		w.walkDeclList(x.Decls)

	case *ast.Package:
		for _, f := range x.Files {
			w.walkOther(f)
		}
	}
}
