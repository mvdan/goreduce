// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import "go/ast"

// TODO: can we do better than every func iterating over the AST on its
// own?

// TODO: recurse into nested blocks

// TODO: go/types could likely catch compile errors that we produced
// without having to call the go tool. Study whether inferring that
// ourselves for common cases (e.g. removing var declarations) would
// save time.

// TODO: is this powerful and versatile enough?
// Some ideas:
// * It doesn't have the ast for the whole package/file
// * go/types info could be useful
// * Work on x/tools/go/ssa, even?
type changeFunc func(*ast.BlockStmt) []*ast.BlockStmt

// TODO: don't generate new ASTs all at once, use something else like a
// chan
func allChanges(orig *ast.BlockStmt) (bs []*ast.BlockStmt) {
	for _, f := range []changeFunc{
		// more aggressive changes first to try and speed it up
		removeStmt,
		bypassIf,
		bypassElse,
	} {
		bs = append(bs, f(orig)...)
	}
	return
}

// xs; y; zs -> xs; zs
func removeStmt(orig *ast.BlockStmt) []*ast.BlockStmt {
	bs := make([]*ast.BlockStmt, len(orig.List))
	for i := range orig.List {
		b := &ast.BlockStmt{}
		bs[i], *b = b, *orig
		b.List = make([]ast.Stmt, len(orig.List)-1)
		copy(b.List, orig.List[:i])
		copy(b.List[i:], orig.List[i+1:])
	}
	return bs
}

// if xs { ys } -> ys
func bypassIf(orig *ast.BlockStmt) []*ast.BlockStmt {
	bs := []*ast.BlockStmt{}
	for i, stmt := range orig.List {
		ifStmt, ok := stmt.(*ast.IfStmt)
		if !ok {
			continue
		}
		b := &ast.BlockStmt{}
		bs, *b = append(bs, b), *orig
		b.List = make([]ast.Stmt, len(orig.List))
		copy(b.List, orig.List)
		b.List[i] = ifStmt.Body
	}
	return bs
}

// if xs { ys } else z -> z
func bypassElse(orig *ast.BlockStmt) []*ast.BlockStmt {
	bs := []*ast.BlockStmt{}
	for i, stmt := range orig.List {
		ifStmt, ok := stmt.(*ast.IfStmt)
		if !ok || ifStmt.Else == nil {
			continue
		}
		b := &ast.BlockStmt{}
		bs, *b = append(bs, b), *orig
		b.List = make([]ast.Stmt, len(orig.List))
		copy(b.List, orig.List)
		b.List[i] = ifStmt.Else
	}
	return bs
}
