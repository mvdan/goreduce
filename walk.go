// Heavily modivied version of Go's src/go/ast/walk.go

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import (
	"go/ast"
	"go/token"
)

func (r *reducer) walkIdentList(list []*ast.Ident) {
	for _, x := range list {
		r.walkOther(x)
	}
}

func (r *reducer) walkExprList(list []ast.Expr) {
	for i := range list {
		r.walkExpr(&list[i])
	}
}

func (r *reducer) walkStmtList(list *[]ast.Stmt) {
	orig := *list
	// RULE: remove each one of the statements
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
		l := make([]ast.Stmt, len(orig)-1)
		copy(l, orig[:i])
		copy(l[i:], orig[i+1:])
		*list = l
		if r.okChange() {
			return
		}
	}
	*list = orig
	for i := range orig {
		r.walkStmt(&orig[i])
	}
}

func (r *reducer) walkDeclList(list []ast.Decl) {
	for _, x := range list {
		r.walkOther(x)
	}
}

func (r *reducer) walkStmt(stmt *ast.Stmt) {
	r.stmt = stmt
	r.expr = nil
	r.walk(*stmt)
}

func (r *reducer) walkExpr(expr *ast.Expr) {
	r.stmt = nil
	r.expr = expr
	r.walk(*expr)
}

func (r *reducer) walkOther(node ast.Node) {
	r.stmt = nil
	r.expr = nil
	r.walk(node)
}

func (r *reducer) walk(node ast.Node) {
	if r.didChange {
		return
	}
	switch x := node.(type) {
	// Fields
	case *ast.Field:
		r.walkIdentList(x.Names)
		r.walkExpr(&x.Type)
		if x.Tag != nil {
			r.walkOther(x.Tag)
		}

	case *ast.FieldList:
		for _, f := range x.List {
			r.walkOther(f)
		}

	// Expressions
	case *ast.BasicLit:
		r.reduceLit(x)

	case *ast.Ident:
		// nothing to do

	case *ast.Ellipsis:
		if x.Elt != nil {
			r.walkExpr(&x.Elt)
		}

	case *ast.FuncLit:
		r.walkOther(x.Type)
		r.walkOther(x.Body)

	case *ast.CompositeLit:
		if x.Type != nil {
			r.walkExpr(&x.Type)
		}
		r.walkExprList(x.Elts)

	case *ast.ParenExpr:
		r.walkExpr(&x.X)

	case *ast.SelectorExpr:
		r.walkExpr(&x.X)
		r.walkOther(x.Sel)

	case *ast.IndexExpr:
		r.walkExpr(&x.X)
		r.walkExpr(&x.Index)

	case *ast.SliceExpr:
		r.reduceSlice(x)
		r.walkExpr(&x.X)
		if x.Low != nil {
			r.walkExpr(&x.Low)
		}
		if x.High != nil {
			r.walkExpr(&x.High)
		}
		if x.Max != nil {
			r.walkExpr(&x.Max)
		}

	case *ast.TypeAssertExpr:
		r.walkExpr(&x.X)
		if x.Type != nil {
			r.walkExpr(&x.Type)
		}

	case *ast.CallExpr:
		r.walkExpr(&x.Fun)
		r.walkExprList(x.Args)

	case *ast.StarExpr:
		r.walkExpr(&x.X)

	case *ast.UnaryExpr:
		r.walkExpr(&x.X)

	case *ast.BinaryExpr:
		r.reduceBinary(x)
		r.walkExpr(&x.X)
		r.walkExpr(&x.Y)

	case *ast.KeyValueExpr:
		r.walkExpr(&x.Key)
		r.walkExpr(&x.Value)

	// Types
	case *ast.ArrayType:
		if x.Len != nil {
			r.walkExpr(&x.Len)
		}
		r.walkExpr(&x.Elt)

	case *ast.StructType:
		r.walkOther(x.Fields)

	case *ast.FuncType:
		if x.Params != nil {
			r.walkOther(x.Params)
		}
		if x.Results != nil {
			r.walkOther(x.Results)
		}

	case *ast.InterfaceType:
		r.walkOther(x.Methods)

	case *ast.MapType:
		r.walkExpr(&x.Key)
		r.walkExpr(&x.Value)

	case *ast.ChanType:
		r.walkExpr(&x.Value)

	// Statements
	case *ast.DeclStmt:
		r.walkOther(x.Decl)

	case *ast.LabeledStmt:
		r.walkOther(x.Label)
		r.walkStmt(&x.Stmt)

	case *ast.ExprStmt:
		r.walkExpr(&x.X)

	case *ast.SendStmt:
		r.walkExpr(&x.Chan)
		r.walkExpr(&x.Value)

	case *ast.IncDecStmt:
		r.walkExpr(&x.X)

	case *ast.AssignStmt:
		r.walkExprList(x.Lhs)
		r.walkExprList(x.Rhs)

	case *ast.GoStmt:
		r.walkOther(x.Call)

	case *ast.DeferStmt:
		r.bypassDefer(x)
		r.walkOther(x.Call)

	case *ast.ReturnStmt:
		r.walkExprList(x.Results)

	case *ast.BranchStmt:
		if x.Label != nil {
			r.walkOther(x.Label)
		}

	case *ast.BlockStmt:
		r.walkStmtList(&x.List)

	case *ast.IfStmt:
		r.bypassIf(x)
		if x.Init != nil {
			r.walkStmt(&x.Init)
		}
		r.walkExpr(&x.Cond)
		r.walkOther(x.Body)
		if x.Else != nil {
			r.walkStmt(&x.Else)
		}

	case *ast.CaseClause:
		r.walkExprList(x.List)
		r.walkStmtList(&x.Body)

	case *ast.SwitchStmt:
		if x.Init != nil {
			r.walkStmt(&x.Init)
		}
		if x.Tag != nil {
			r.walkOther(x.Tag)
		}
		r.walkOther(x.Body)

	case *ast.TypeSwitchStmt:
		if x.Init != nil {
			r.walkStmt(&x.Init)
		}
		r.walkStmt(&x.Assign)
		r.walkOther(x.Body)

	case *ast.CommClause:
		if x.Comm != nil {
			r.walkStmt(&x.Comm)
		}
		r.walkStmtList(&x.Body)

	case *ast.SelectStmt:
		r.walkOther(x.Body)

	case *ast.ForStmt:
		if x.Init != nil {
			r.walkStmt(&x.Init)
		}
		if x.Cond != nil {
			r.walkExpr(&x.Cond)
		}
		if x.Post != nil {
			r.walkStmt(&x.Post)
		}
		r.walkOther(x.Body)

	case *ast.RangeStmt:
		if x.Key != nil {
			r.walkExpr(&x.Key)
		}
		if x.Value != nil {
			r.walkExpr(&x.Value)
		}
		r.walkExpr(&x.X)
		r.walkOther(x.Body)

	// Declarations
	case *ast.ImportSpec:
		if x.Name != nil {
			r.walkOther(x.Name)
		}
		r.walkOther(x.Path)

	case *ast.ValueSpec:
		r.walkIdentList(x.Names)
		if x.Type != nil {
			r.walkExpr(&x.Type)
		}
		r.walkExprList(x.Values)

	case *ast.TypeSpec:
		r.walkOther(x.Name)
		r.walkExpr(&x.Type)

	case *ast.GenDecl:
		for _, s := range x.Specs {
			r.walkOther(s)
		}

	case *ast.FuncDecl:
		if x.Recv != nil {
			r.walkOther(x.Recv)
		}
		r.walkOther(x.Name)
		r.walkOther(x.Type)
		if x.Body != nil {
			r.walkOther(x.Body)
		}

	// Files and packages
	case *ast.File:
		r.walkOther(x.Name)
		r.walkDeclList(x.Decls)

	case *ast.Package:
		for _, f := range x.Files {
			r.walkOther(f)
		}
	}
}
