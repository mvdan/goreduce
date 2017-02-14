// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

const mainFile = "goreduce_main.go"

var (
	mainTmpl = template.Must(template.New("main").Parse(`package main

func main() {
	{{ . }}()
}
`))
	rawPrinter = printer.Config{Mode: printer.RawFormat}
)

type reducer struct {
	dir     string
	matchRe *regexp.Regexp

	fset     *token.FileSet
	pkg      *ast.Package
	files    []*ast.File
	file     *ast.File
	funcDecl *ast.FuncDecl
	origMain *ast.FuncDecl

	tconf types.Config
	info  *types.Info

	numUses map[types.Object]int

	outBin  string
	goArgs  []string
	dstFile *os.File

	didChange bool
	stmt      *ast.Stmt
	expr      *ast.Expr
}

var errNoReduction = fmt.Errorf("could not reduce program")

func reduce(dir, funcName, matchStr string, bflags ...string) error {
	r := &reducer{dir: dir}
	tdir, err := ioutil.TempDir("", "goreduce")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tdir)
	r.tconf.Importer = importer.Default()
	if r.matchRe, err = regexp.Compile(matchStr); err != nil {
		return err
	}
	r.fset = token.NewFileSet()
	pkgs, err := parser.ParseDir(r.fset, r.dir, nil, 0)
	if err != nil {
		return err
	}
	if len(pkgs) != 1 {
		return fmt.Errorf("expected 1 package, got %d", len(pkgs))
	}
	for _, pkg := range pkgs {
		r.pkg = pkg
	}
	for _, file := range r.pkg.Files {
		r.files = append(r.files, file)
	}
	r.file, r.funcDecl = findFunc(r.files, funcName)
	if r.file == nil {
		return fmt.Errorf("top-level func %s does not exist", funcName)
	}
	tfnames := make([]string, 0, len(r.files)+1)
	for _, file := range r.files {
		fname := r.fset.Position(file.Pos()).Filename
		tfname := filepath.Join(tdir, filepath.Base(fname))
		f, err := os.Create(tfname)
		if err != nil {
			return nil
		}
		if file.Name.Name == "main" {
			if fd := delFunc(file, "main"); fd != nil && file == r.file {
				r.origMain = fd
			}
		} else {
			file.Name.Name = "main"
		}
		if err := rawPrinter.Fprint(f, r.fset, file); err != nil {
			return err
		}
		if file == r.file {
			r.dstFile = f
			defer r.dstFile.Close()
		} else if err := f.Close(); err != nil {
			return err
		}
		tfnames = append(tfnames, tfname)
	}
	mfname := filepath.Join(tdir, mainFile)
	mf, err := os.Create(mfname)
	if err != nil {
		return err
	}
	if err := mainTmpl.Execute(mf, funcName); err != nil {
		return err
	}
	if err := mf.Close(); err != nil {
		return err
	}
	tfnames = append(tfnames, mfname)
	r.outBin = filepath.Join(tdir, "bin")
	r.goArgs = []string{"build", "-o", r.outBin}
	r.goArgs = append(r.goArgs, buildFlags...)
	r.goArgs = append(r.goArgs, bflags...)
	r.goArgs = append(r.goArgs, tfnames...)
	// Check that the output matches before we apply any changes
	if err := r.checkRun(); err != nil {
		return err
	}
	if anyChanges := r.reduceLoop(); !anyChanges {
		return errNoReduction
	}
	fname := r.fset.Position(r.file.Pos()).Filename
	f, err := os.Create(fname)
	if err != nil {
		return err
	}
	r.file.Name.Name = r.pkg.Name
	if r.origMain != nil {
		r.file.Decls = append(r.file.Decls, r.origMain)
	}
	if err := printer.Fprint(f, r.fset, r.file); err != nil {
		return err
	}
	return f.Close()
}

func (r *reducer) logChange(node ast.Node, format string, a ...interface{}) {
	if *verbose {
		pos := r.fset.Position(node.Pos())
		fmt.Fprintf(os.Stderr, "%s:%d: %s\n", pos.Filename, pos.Line,
			fmt.Sprintf(format, a...))
	}
}

func (r *reducer) checkRun() error {
	if err := r.buildAndRun(); err == nil {
		return fmt.Errorf("expected an error to occur")
	} else if s := err.Error(); !r.matchRe.MatchString(s) {
		return fmt.Errorf("error does not match:\n%s", s)
	}
	return nil
}

func (r *reducer) okChange() bool {
	if r.didChange {
		return false
	}
	// go/types catches most compile errors before writing
	// to disk and running the go tool. Since quite a lot of
	// changes are nonsensical, this is often a big win.
	if _, err := r.tconf.Check(r.dir, r.fset, r.files, nil); err != nil {
		if terr, ok := err.(types.Error); ok && terr.Soft {
			println("unexpected go/types soft error: " + terr.Msg)
		}
		return false
	}
	if err := r.dstFile.Truncate(0); err != nil {
		return false
	}
	if _, err := r.dstFile.Seek(0, 0); err != nil {
		return false
	}
	if err := rawPrinter.Fprint(r.dstFile, r.fset, r.file); err != nil {
		return false
	}
	if err := r.checkRun(); err != nil {
		return false
	}
	// Reduction worked
	r.didChange = true
	return true
}

func (r *reducer) reduceLoop() (anyChanges bool) {
	r.info = &types.Info{
		Defs: make(map[*ast.Ident]types.Object),
		Uses: make(map[*ast.Ident]types.Object),
	}
	for {
		if _, err := r.tconf.Check(r.dir, r.fset, r.files, r.info); err != nil {
			panic("types.Check should not error here")
		}
		r.fillUses()
		r.didChange = false
		r.walk(r.file, r.reduceNode)
		if !r.didChange {
			return
		}
		anyChanges = true
	}
}

func (r *reducer) fillUses() {
	r.numUses = make(map[types.Object]int)
	for _, obj := range r.info.Uses {
		if pkg := obj.Pkg(); pkg == nil || pkg.Name() != "main" {
			// builtin or declared outside of our pkg
			continue
		}
		switch obj.(type) {
		case *types.PkgName, *types.Var:
			r.numUses[obj]++
		}
	}
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

func delFunc(file *ast.File, name string) *ast.FuncDecl {
	for i, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if ok && fd.Name.Name == name {
			file.Decls = append(file.Decls[:i], file.Decls[i+1:]...)
			return fd
		}
	}
	return nil
}

func (r *reducer) buildAndRun() error {
	cmd := exec.Command("go", r.goArgs...)
	if out, err := cmd.CombinedOutput(); err != nil {
		if strings.HasPrefix(err.Error(), "exit status") {
			return errors.New(string(out))
		}
		return err
	}
	if out, err := exec.Command(r.outBin).CombinedOutput(); err != nil {
		if strings.HasPrefix(err.Error(), "exit status") {
			return errors.New(string(out))
		}
		return err
	}
	return nil
}
