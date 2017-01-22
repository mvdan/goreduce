// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/printer"
	"html/template"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"golang.org/x/tools/go/loader"

	"github.com/kisielk/gotool"
)

const (
	testFile = "reduce_test.go"
	testName = "TestReduce"
)

var testTmpl = template.Must(template.New("test").Parse(`` +
	`package {{ .Pkg }}

import "testing"

func {{ .TestName }}(t *testing.T) {
	{{ .Func }}()
}
`))

func emptyFile(f *os.File) error {
	if err := f.Truncate(0); err != nil {
		return err
	}
	_, err := f.Seek(0, 0)
	return err
}

func writeTest(f *os.File, pkgName, funcName string) error {
	if err := emptyFile(f); err != nil {
		return err
	}
	return testTmpl.Execute(f, struct {
		Pkg, TestName, Func string
	}{
		Pkg:      pkgName,
		TestName: testName,
		Func:     funcName,
	})
}

func reduce(funcName, matchStr string) error {
	if matchStr == "" {
		return fmt.Errorf("match regexp cannot be empty")
	}
	r := &reducer{}
	var err error
	if r.matchRe, err = regexp.Compile(matchStr); err != nil {
		return err
	}
	paths := gotool.ImportPaths([]string{"."})
	if _, err := r.FromArgs(paths, false); err != nil {
		return err
	}
	prog, err := r.Load()
	if err != nil {
		return err
	}
	pkgInfos := prog.InitialPackages()
	if len(pkgInfos) != 1 {
		return fmt.Errorf("expected 1 package, got %d", len(pkgInfos))
	}
	r.PackageInfo = pkgInfos[0]
	pkg := r.PackageInfo.Pkg
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
	// Check that it compiles and the output matches before we apply
	// any changes
	if err := writeTest(tf, pkg.Name(), funcName); err != nil {
		return err
	}
	if err := r.checkTest(); err != nil {
		return err
	}
	r.file, r.funcDecl = findFunc(r.PackageInfo.Files, funcName)
	fname := r.Fset.Position(r.file.Pos()).Filename
	if r.srcFile, err = os.Create(fname); err != nil {
		return err
	}
	defer r.srcFile.Close()
	for {
		if err := r.step(); err == errNoChange {
			break // we're done
		} else if err != nil {
			return err
		}
	}
	return nil
}

type reducer struct {
	loader.Config
	*loader.PackageInfo
	matchRe  *regexp.Regexp
	file     *ast.File
	funcDecl *ast.FuncDecl
	srcFile  *os.File
}

func matchError(matchRe *regexp.Regexp, err error) error {
	if err == nil {
		return fmt.Errorf("expected an error to occur")
	}
	if s := err.Error(); !matchRe.MatchString(s) {
		return fmt.Errorf("error does not match:\n%s", s)
	}
	return nil
}

func (r *reducer) checkTest() error {
	return matchError(r.matchRe, runTest())
}

var errNoChange = fmt.Errorf("no reduction to apply")

func (r *reducer) step() error {
	block := r.funcDecl.Body
	for _, b := range removeStmt(block) {
		r.funcDecl.Body = b
		if err := emptyFile(r.srcFile); err != nil {
			return err
		}
		if err := printer.Fprint(r.srcFile, r.Fset, r.file); err != nil {
			return err
		}
		if err := r.checkTest(); err == nil {
			// Reduction worked
			return nil
		}
	}
	// Nothing worked, return to original state
	r.funcDecl.Body = block
	if err := emptyFile(r.srcFile); err != nil {
		return err
	}
	printer.Fprint(r.srcFile, r.Fset, r.file)
	return errNoChange
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
			funcDecl, ok := decl.(*ast.FuncDecl)
			if ok && funcDecl.Name.Name == name {
				return file, funcDecl
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
