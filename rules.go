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

func (r *reducer) changeStmt(stmt ast.Stmt) bool {
	orig := *r.stmt
	*r.stmt = stmt
	if r.okChange() {
		return true
	}
	*r.stmt = orig
	return false
}

func (r *reducer) bypassIf(ifs *ast.IfStmt) {
	if r.changeStmt(ifs.Body) {
		return
	}
	if ifs.Else != nil && r.changeStmt(ifs.Else) {
		return
	}
}
