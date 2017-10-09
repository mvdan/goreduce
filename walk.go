// Heavily modivied version of Go's src/go/ast/walk.go

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import "go/ast"

type walker struct {
	fn    func(v interface{}) bool
	queue []interface{}
	file  *ast.File
}

func (w *walker) walk(v interface{}, fn func(interface{}) bool) {
	w.queue = w.queue[:0]
	w.fn = fn
	w.walkOther(v)
	for len(w.queue) > 0 {
		v := w.queue[0]
		w.queue = w.queue[1:]
		w.walkSingle(v)
	}
}

func (w *walker) walkIdentList(list []*ast.Ident) {
	for _, x := range list {
		w.walkOther(x)
	}
}

func (w *walker) walkExprList(list []ast.Expr) {
	for _, x := range list {
		w.walkOther(x)
	}
}

func (w *walker) walkStmtList(list *[]ast.Stmt) {
	if len(*list) > 0 {
		w.walkOther(list)
	}
}

func (w *walker) walkDeclList(list []ast.Decl) {
	for _, x := range list {
		w.walkOther(x)
	}
}

func (w *walker) walkOther(v interface{}) {
	w.queue = append(w.queue, v)
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
			w.walkOther(l[i])
		}
	// Fields
	case *ast.Field:
		w.walkIdentList(x.Names)
		w.walkOther(x.Type)
		if x.Tag != nil {
			w.walkOther(x.Tag)
		}

	case *ast.FieldList:
		for _, f := range x.List {
			w.walkOther(f)
		}

	// Expressions
	case *ast.Ellipsis:
		if x.Elt != nil {
			w.walkOther(x.Elt)
		}

	case *ast.FuncLit:
		w.walkOther(x.Type)
		w.walkOther(x.Body)

	case *ast.CompositeLit:
		if x.Type != nil {
			w.walkOther(x.Type)
		}
		w.walkExprList(x.Elts)

	case *ast.ParenExpr:
		w.walkOther(x.X)

	case *ast.SelectorExpr:
		w.walkOther(x.X)
		w.walkOther(x.Sel)

	case *ast.IndexExpr:
		w.walkOther(x.X)
		w.walkOther(x.Index)

	case *ast.SliceExpr:
		w.walkOther(x.X)
		if x.Low != nil {
			w.walkOther(x.Low)
		}
		if x.High != nil {
			w.walkOther(x.High)
		}
		if x.Max != nil {
			w.walkOther(x.Max)
		}

	case *ast.TypeAssertExpr:
		w.walkOther(x.X)
		if x.Type != nil {
			w.walkOther(x.Type)
		}

	case *ast.CallExpr:
		w.walkOther(x.Fun)
		w.walkExprList(x.Args)

	case *ast.StarExpr:
		w.walkOther(x.X)

	case *ast.UnaryExpr:
		w.walkOther(x.X)

	case *ast.BinaryExpr:
		w.walkOther(x.X)
		w.walkOther(x.Y)

	case *ast.KeyValueExpr:
		w.walkOther(x.Key)
		w.walkOther(x.Value)

	// Types
	case *ast.ArrayType:
		if x.Len != nil {
			w.walkOther(x.Len)
		}
		w.walkOther(x.Elt)

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
		w.walkOther(x.Key)
		w.walkOther(x.Value)

	case *ast.ChanType:
		w.walkOther(x.Value)

	// Statements
	case *ast.DeclStmt:
		w.walkOther(x.Decl)

	case *ast.LabeledStmt:
		w.walkOther(x.Label)
		w.walkOther(x.Stmt)

	case *ast.ExprStmt:
		w.walkOther(x.X)

	case *ast.SendStmt:
		w.walkOther(x.Chan)
		w.walkOther(x.Value)

	case *ast.IncDecStmt:
		w.walkOther(x.X)

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
			w.walkOther(x.Init)
		}
		w.walkOther(x.Cond)
		w.walkOther(x.Body)
		if x.Else != nil {
			w.walkOther(x.Else)
		}

	case *ast.CaseClause:
		w.walkExprList(x.List)
		w.walkStmtList(&x.Body)

	case *ast.SwitchStmt:
		if x.Init != nil {
			w.walkOther(x.Init)
		}
		if x.Tag != nil {
			w.walkOther(x.Tag)
		}
		w.walkOther(x.Body)

	case *ast.TypeSwitchStmt:
		if x.Init != nil {
			w.walkOther(x.Init)
		}
		w.walkOther(x.Assign)
		w.walkOther(x.Body)

	case *ast.CommClause:
		if x.Comm != nil {
			w.walkOther(x.Comm)
		}
		w.walkStmtList(&x.Body)

	case *ast.SelectStmt:
		w.walkOther(x.Body)

	case *ast.ForStmt:
		if x.Init != nil {
			w.walkOther(x.Init)
		}
		if x.Cond != nil {
			w.walkOther(x.Cond)
		}
		if x.Post != nil {
			w.walkOther(x.Post)
		}
		w.walkOther(x.Body)

	case *ast.RangeStmt:
		if x.Key != nil {
			w.walkOther(x.Key)
		}
		if x.Value != nil {
			w.walkOther(x.Value)
		}
		w.walkOther(x.X)
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
			w.walkOther(x.Type)
		}
		w.walkExprList(x.Values)

	case *ast.TypeSpec:
		w.walkOther(x.Name)
		w.walkOther(x.Type)

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
		w.file = x
		w.walkOther(x.Name)
		w.walkDeclList(x.Decls)

	case *ast.Package:
		for _, f := range x.Files {
			w.walkOther(f)
		}
	}
}
