// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import (
	"go/ast"
	"go/token"
)

// TODO: we use go/types to catch compile errors before writing to disk
// and running the go tool. Study whether it's worth anticipating some
// of the common cases (e.g. removing var declarations) to save time.

// TODO: is this powerful and versatile enough?
// Some ideas:
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
	// RULE: bypass to if or else branches
	if r.changeStmt(ifs.Body) {
		return
	}
	if ifs.Else != nil && r.changeStmt(ifs.Else) {
		return
	}
}

func (r *reducer) reduceLit(l *ast.BasicLit) {
	orig := l.Value
	okValue := func(val string) bool {
		if l.Value == val {
			return false
		}
		l.Value = val
		if r.okChange() {
			return true
		}
		l.Value = orig
		return false
	}
	switch l.Kind {
	case token.STRING:
		switch {
		case okValue(`""`):
		}
	}
}
