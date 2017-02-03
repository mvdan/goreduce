// Heavily modivied version of Go's src/go/ast/walk.go

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import (
	"go/ast"
)

func (r *reducer) walkIdentList(list []*ast.Ident) {
	for _, x := range list {
		r.walk(x)
	}
}

func (r *reducer) walkExprList(list []ast.Expr) {
	for _, x := range list {
		r.walk(x)
	}
}

func (r *reducer) walkStmtList(list *[]ast.Stmt) {
	orig := *list
	// RULE: remove each one of the statements
	for i := range orig {
		l := make([]ast.Stmt, len(orig)-1)
		copy(l, orig[:i])
		copy(l[i:], orig[i+1:])
		*list = l
		if r.okChange() {
			return
		}
	}
	*list = orig
	for i, x := range orig {
		r.stmt = &orig[i]
		r.walk(x)
	}
}

func (r *reducer) walkDeclList(list []ast.Decl) {
	for _, x := range list {
		r.walk(x)
	}
}

func (r *reducer) walk(node ast.Node) {
	switch x := node.(type) {
	// Fields
	case *ast.Field:
		r.walkIdentList(x.Names)
		r.walk(x.Type)
		if x.Tag != nil {
			r.walk(x.Tag)
		}

	case *ast.FieldList:
		for _, f := range x.List {
			r.walk(f)
		}

	// Expressions
	case *ast.Ident, *ast.BasicLit:
		// nothing to do

	case *ast.Ellipsis:
		if x.Elt != nil {
			r.walk(x.Elt)
		}

	case *ast.FuncLit:
		r.walk(x.Type)
		r.walk(x.Body)

	case *ast.CompositeLit:
		if x.Type != nil {
			r.walk(x.Type)
		}
		r.walkExprList(x.Elts)

	case *ast.ParenExpr:
		r.walk(x.X)

	case *ast.SelectorExpr:
		r.walk(x.X)
		r.walk(x.Sel)

	case *ast.IndexExpr:
		r.walk(x.X)
		r.walk(x.Index)

	case *ast.SliceExpr:
		r.walk(x.X)
		if x.Low != nil {
			r.walk(x.Low)
		}
		if x.High != nil {
			r.walk(x.High)
		}
		if x.Max != nil {
			r.walk(x.Max)
		}

	case *ast.TypeAssertExpr:
		r.walk(x.X)
		if x.Type != nil {
			r.walk(x.Type)
		}

	case *ast.CallExpr:
		r.walk(x.Fun)
		r.walkExprList(x.Args)

	case *ast.StarExpr:
		r.walk(x.X)

	case *ast.UnaryExpr:
		r.walk(x.X)

	case *ast.BinaryExpr:
		r.walk(x.X)
		r.walk(x.Y)

	case *ast.KeyValueExpr:
		r.walk(x.Key)
		r.walk(x.Value)

	// Types
	case *ast.ArrayType:
		if x.Len != nil {
			r.walk(x.Len)
		}
		r.walk(x.Elt)

	case *ast.StructType:
		r.walk(x.Fields)

	case *ast.FuncType:
		if x.Params != nil {
			r.walk(x.Params)
		}
		if x.Results != nil {
			r.walk(x.Results)
		}

	case *ast.InterfaceType:
		r.walk(x.Methods)

	case *ast.MapType:
		r.walk(x.Key)
		r.walk(x.Value)

	case *ast.ChanType:
		r.walk(x.Value)

	// Statements
	case *ast.DeclStmt:
		r.walk(x.Decl)

	case *ast.EmptyStmt:
		// nothing to do

	case *ast.LabeledStmt:
		r.walk(x.Label)
		r.walk(x.Stmt)

	case *ast.ExprStmt:
		r.walk(x.X)

	case *ast.SendStmt:
		r.walk(x.Chan)
		r.walk(x.Value)

	case *ast.IncDecStmt:
		r.walk(x.X)

	case *ast.AssignStmt:
		r.walkExprList(x.Lhs)
		r.walkExprList(x.Rhs)

	case *ast.GoStmt:
		r.walk(x.Call)

	case *ast.DeferStmt:
		r.walk(x.Call)

	case *ast.ReturnStmt:
		r.walkExprList(x.Results)

	case *ast.BranchStmt:
		if x.Label != nil {
			r.walk(x.Label)
		}

	case *ast.BlockStmt:
		r.walkStmtList(&x.List)

	case *ast.IfStmt:
		r.bypassIf(x)
		if x.Init != nil {
			r.walk(x.Init)
		}
		r.walk(x.Cond)
		r.walk(x.Body)
		if x.Else != nil {
			r.walk(x.Else)
		}

	case *ast.CaseClause:
		r.walkExprList(x.List)
		r.walkStmtList(&x.Body)

	case *ast.SwitchStmt:
		if x.Init != nil {
			r.walk(x.Init)
		}
		if x.Tag != nil {
			r.walk(x.Tag)
		}
		r.walk(x.Body)

	case *ast.TypeSwitchStmt:
		if x.Init != nil {
			r.walk(x.Init)
		}
		r.walk(x.Assign)
		r.walk(x.Body)

	case *ast.CommClause:
		if x.Comm != nil {
			r.walk(x.Comm)
		}
		r.walkStmtList(&x.Body)

	case *ast.SelectStmt:
		r.walk(x.Body)

	case *ast.ForStmt:
		if x.Init != nil {
			r.walk(x.Init)
		}
		if x.Cond != nil {
			r.walk(x.Cond)
		}
		if x.Post != nil {
			r.walk(x.Post)
		}
		r.walk(x.Body)

	case *ast.RangeStmt:
		if x.Key != nil {
			r.walk(x.Key)
		}
		if x.Value != nil {
			r.walk(x.Value)
		}
		r.walk(x.X)
		r.walk(x.Body)

	// Declarations
	case *ast.ImportSpec:
		if x.Name != nil {
			r.walk(x.Name)
		}
		r.walk(x.Path)

	case *ast.ValueSpec:
		r.walkIdentList(x.Names)
		if x.Type != nil {
			r.walk(x.Type)
		}
		r.walkExprList(x.Values)

	case *ast.TypeSpec:
		r.walk(x.Name)
		r.walk(x.Type)

	case *ast.GenDecl:
		for _, s := range x.Specs {
			r.walk(s)
		}

	case *ast.FuncDecl:
		if x.Recv != nil {
			r.walk(x.Recv)
		}
		r.walk(x.Name)
		r.walk(x.Type)
		if x.Body != nil {
			r.walk(x.Body)
		}

	// Files and packages
	case *ast.File:
		r.walk(x.Name)
		r.walkDeclList(x.Decls)

	case *ast.Package:
		for _, f := range x.Files {
			r.walk(f)
		}
	}
}
