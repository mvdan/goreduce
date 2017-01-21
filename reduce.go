// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import (
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/printer"
	"html/template"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/tools/go/loader"

	"github.com/kisielk/gotool"
)

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "Func name missing: goreduce funcName")
		os.Exit(1)
	}
	if err := reduce(args[0]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

const (
	testFile = "reduce_test.go"
	testName = "TestReduce"
)

var testTmpl = template.Must(template.New("test").Parse(`` +
	`package {{ .Pkg }}

import "testing"

func {{ .TestName }}(t *testing.T) {
	if {{ .Expr }}() {
		t.Fail()
	}
}
`))

func emptyFile(f *os.File) error {
	if err := f.Truncate(0); err != nil {
		return err
	}
	_, err := f.Seek(0, 0)
	return err
}

func writeTest(f *os.File, pkgName, expr string) error {
	if err := emptyFile(f); err != nil {
		return err
	}
	return testTmpl.Execute(f, struct {
		Pkg, TestName, Expr string
	}{
		Pkg:      pkgName,
		TestName: testName,
		Expr:     expr,
	})
}

func reduce(funcName string) error {
	paths := gotool.ImportPaths([]string{"."})
	conf := loader.Config{}
	if _, err := conf.FromArgs(paths, false); err != nil {
		return err
	}
	prog, err := conf.Load()
	if err != nil {
		return err
	}
	pkgInfos := prog.InitialPackages()
	if len(pkgInfos) != 1 {
		return fmt.Errorf("expected 1 package, got %d", len(pkgInfos))
	}
	pkgInfo := pkgInfos[0]
	pkg := pkgInfo.Pkg
	if pkg.Scope().Lookup(funcName) == nil {
		return fmt.Errorf("top-level func %s does not exist", funcName)
	}
	tf, err := os.Create(testFile)
	if err != nil {
		return err
	}
	defer func() {
		tf.Close()
		os.Remove(testFile)
	}()
	// Check that it compiles and the func returns true, meaning
	// that it's still reproducing the issue.
	if err := writeTest(tf, pkg.Name(), funcName); err != nil {
		return err
	}
	if err := runTest(); err == nil {
		return fmt.Errorf("expected test to fail")
	}
	if err := writeTest(tf, pkg.Name(), "!"+funcName); err != nil {
		return err
	}
	if err := runTest(); err != nil {
		return err
	}
	file, fd := findFunc(pkgInfo.Files, funcName)
	fname := conf.Fset.Position(file.Pos()).Filename
	f, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer f.Close()
	block := fd.Body
	for _, b := range removeStmt(block) {
		fd.Body = b
		if err := emptyFile(f); err != nil {
			return err
		}
		if err := printer.Fprint(f, conf.Fset, file); err != nil {
			return err
		}
		if err := runTest(); err == nil {
			// Reduction worked, exit
			return nil
		}
	}
	// Nothing worked, return to original state
	fd.Body = block
	printer.Fprint(f, conf.Fset, file)
	return nil
}

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

func findFunc(files []*ast.File, name string) (*ast.File, *ast.FuncDecl) {
	for _, file := range files {
		for _, decl := range file.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if ok && fd.Name.Name == name {
				return file, fd
			}
		}
	}
	return nil, nil
}

func runTest() error {
	cmd := exec.Command("go", "test", "-run", "^"+testName+"$")
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	if strings.HasPrefix(err.Error(), "exit status") {
		return errors.New(strings.TrimSpace(string(out)))
	}
	return err
}
