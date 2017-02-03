// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import "go/ast"

// TODO: we use go/types to catch compile errors before writing to disk
// and running the go tool. Study whether it's worth anticipating some
// of the common cases (e.g. removing var declarations) to save time.

// TODO: is this powerful and versatile enough?
// Some ideas:
// * It doesn't have the ast for the whole package/file
// * go/types info could be useful
// * Work on x/tools/go/ssa, even?
type changeFunc func(*ast.BlockStmt) []*ast.BlockStmt

func (r *reducer) changes(orig *ast.BlockStmt) {
	r.didChange = false
	ast.Walk(r, orig)
	return
}

func (r *reducer) Visit(node ast.Node) ast.Visitor {
	switch x := node.(type) {
	case *ast.BlockStmt:
		r.removeStmt(x)
		r.bypassIf(x)
	}
	return r
}

// xs; y; zs -> xs; zs
func (r *reducer) removeStmt(b *ast.BlockStmt) {
	orig := b.List
	for i := range orig {
		b.List = make([]ast.Stmt, len(orig)-1)
		copy(b.List, orig[:i])
		copy(b.List[i:], orig[i+1:])
		if r.check() == validChange {
			return
		}
		b.List = orig
	}
}

// if xs { ys } -> ys
// if xs { ys } else z -> z
func (r *reducer) bypassIf(b *ast.BlockStmt) {
	orig := b.List
	for i, stmt := range orig {
		ifStmt, ok := stmt.(*ast.IfStmt)
		if !ok {
			continue
		}
		b.List = make([]ast.Stmt, len(orig))
		copy(b.List, orig)
		b.List[i] = ifStmt.Body
		if r.check() == validChange {
			return
		}
		if ifStmt.Else != nil {
			b.List = make([]ast.Stmt, len(orig))
			copy(b.List, orig)
			b.List[i] = ifStmt.Else
			if r.check() == validChange {
				return
			}
		}
		b.List = orig
	}
}
