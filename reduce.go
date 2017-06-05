// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"text/template"
)

var (
	mainTmpl = template.Must(template.New("main").Parse(`package main

func main() {
	{{ . }}()
}
`))
	rawPrinter = printer.Config{Mode: printer.RawFormat}

	fastTest = false
)

type reducer struct {
	dir     string
	logOut  io.Writer
	matchRe *regexp.Regexp

	fset     *token.FileSet
	origFset *token.FileSet
	pkg      *ast.Package
	files    []*ast.File
	file     *ast.File

	tconf types.Config
	info  *types.Info

	useIdents map[types.Object][]*ast.Ident
	revDefs   map[types.Object]*ast.Ident
	parents   map[ast.Node]ast.Node

	outBin string
	goArgs []string
	dstBuf *bytes.Buffer
	toRun  bool

	openFiles map[*ast.File]*os.File

	tries     int
	didChange bool

	deleteUndo func()

	tried map[string]bool

	walker
}

var errNoReduction = fmt.Errorf("could not reduce program")

func reduce(dir, funcName, match string, logOut io.Writer, bflags ...string) error {
	r := &reducer{
		dir:    dir,
		logOut: logOut,
		tried:  make(map[string]bool, 16),
		dstBuf: bytes.NewBuffer(nil),
	}
	tdir, err := ioutil.TempDir("", "goreduce")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tdir)
	if r.matchRe, err = regexp.Compile(match); err != nil {
		return err
	}
	r.fset = token.NewFileSet()
	pkgs, err := parser.ParseDir(r.fset, r.dir, nil, parser.ParseComments)
	if err != nil {
		return err
	}
	if len(pkgs) != 1 {
		return fmt.Errorf("expected 1 package, got %d", len(pkgs))
	}
	for _, pkg := range pkgs {
		r.pkg = pkg
	}
	r.origFset = token.NewFileSet()
	parser.ParseDir(r.origFset, r.dir, nil, 0)
	tfnames := make([]string, 0, len(r.pkg.Files)+1)
	foundFunc := false
	r.toRun = funcName != ""
	var origMain *ast.FuncDecl
	r.openFiles = make(map[*ast.File]*os.File, len(r.pkg.Files))
	for fpath, file := range r.pkg.Files {
		if !foundFunc {
			if fd := findFunc(file, funcName); fd != nil {
				r.file = file
				foundFunc = true
			}
		}
		r.files = append(r.files, file)
		tfname := filepath.Join(tdir, filepath.Base(fpath))
		f, err := os.Create(tfname)
		if err != nil {
			return err
		}
		if r.toRun && funcName != "main" {
			if fd := delFunc(file, "main"); fd != nil && file == r.file {
				origMain = fd
			}
			file.Name.Name = "main"
		}
		if err := rawPrinter.Fprint(f, r.fset, file); err != nil {
			return err
		}
		r.openFiles[file] = f
		defer f.Close()
		tfnames = append(tfnames, tfname)
	}
	if !foundFunc {
		if r.toRun {
			return fmt.Errorf("top-level func %s does not exist", funcName)
		}
		for _, file := range r.pkg.Files {
			r.file = file
		}
	}
	if r.toRun && funcName != "main" {
		mfname := filepath.Join(tdir, "goreduce_main.go")
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
	}
	r.tconf.Importer = importer.Default()
	r.tconf.Error = func(err error) {
		if terr, ok := err.(types.Error); ok && terr.Soft {
			// don't stop type-checking on soft errors
			return
		}
		if !r.toRun {
			return
		}
		panic("types.Check should not error here: " + err.Error())
	}
	r.outBin = filepath.Join(tdir, "bin")
	r.goArgs = []string{"build", "-o", r.outBin}
	r.goArgs = append(r.goArgs, buildFlags...)
	r.goArgs = append(r.goArgs, bflags...)
	r.goArgs = append(r.goArgs, tfnames...)
	// Check that the output matches before we apply any changes
	if !fastTest {
		if err := r.checkRun(); err != nil {
			return err
		}
	}
	r.fillParents()
	if anyChanges := r.reduceLoop(); !anyChanges {
		return errNoReduction
	}
	fname := r.fset.Position(r.file.Pos()).Filename
	f, err := os.Create(fname)
	if err != nil {
		return err
	}
	r.file.Name.Name = r.pkg.Name
	if origMain != nil {
		r.file.Decls = append(r.file.Decls, origMain)
	}
	if err := printer.Fprint(f, r.fset, r.file); err != nil {
		return err
	}
	return f.Close()
}

func (r *reducer) logChange(node ast.Node, format string, a ...interface{}) {
	if *verbose {
		pos := r.origFset.Position(node.Pos())
		times := "1 try"
		if r.tries != 1 {
			times = fmt.Sprintf("%d tries", r.tries)
		}
		fmt.Fprintf(r.logOut, "%s:%d: %s (%s)\n",
			pos.Filename, pos.Line, fmt.Sprintf(format, a...), times)
	}
	r.tries = 0
}

func (r *reducer) checkRun() error {
	out := r.buildAndRun()
	if out == nil {
		return fmt.Errorf("expected an error to occur")
	}
	if !r.matchRe.Match(out) {
		return fmt.Errorf("error does not match:\n%s", string(out))
	}
	return nil
}

func (r *reducer) okChangeNoUndo() bool {
	if r.didChange {
		return false
	}
	r.tries++
	r.dstBuf.Reset()
	if err := rawPrinter.Fprint(r.dstBuf, r.fset, r.file); err != nil {
		return false
	}
	newSrc := r.dstBuf.String()
	// TODO: unique per file?
	if r.tried[newSrc] {
		return false
	}
	r.tried[newSrc] = true
	f := r.openFiles[r.file]
	if err := f.Truncate(0); err != nil {
		return false
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return false
	}
	if _, err := f.Write(r.dstBuf.Bytes()); err != nil {
		return false
	}
	if err := r.checkRun(); err != nil {
		return false
	}
	// Reduction worked
	r.didChange = true
	return true
}

func (r *reducer) okChange() bool {
	if r.okChangeNoUndo() {
		return true
	}
	if r.deleteUndo != nil {
		r.deleteUndo()
		r.deleteUndo = nil
	}
	return false
}

func (r *reducer) reduceLoop() (anyChanges bool) {
	r.info = &types.Info{
		Defs: make(map[*ast.Ident]types.Object),
		Uses: make(map[*ast.Ident]types.Object),
	}
	for {
		// Update type info after the AST changes
		r.tconf.Check(r.dir, r.fset, r.files, r.info)
		r.fillObjs()

		r.didChange = false
		r.walk(r.pkg, r.reduceNode)
		if !r.didChange {
			if *verbose {
				fmt.Fprintf(r.logOut, "gave up after %d final tries\n", r.tries)
			}
			return
		}
		anyChanges = true
	}
}

func (r *reducer) fillObjs() {
	r.revDefs = make(map[types.Object]*ast.Ident, len(r.info.Defs))
	for id, obj := range r.info.Defs {
		if obj == nil {
			continue
		}
		r.revDefs[obj] = id
	}
	r.useIdents = make(map[types.Object][]*ast.Ident, len(r.info.Uses)/2)
	for id, obj := range r.info.Uses {
		if pkg := obj.Pkg(); pkg == nil || pkg.Name() != "main" {
			// builtin or declared outside of our pkg
			continue
		}
		r.useIdents[obj] = append(r.useIdents[obj], id)
	}
}

func (r *reducer) fillParents() {
	r.parents = make(map[ast.Node]ast.Node)
	stack := make([]ast.Node, 1, 32)
	ast.Inspect(r.pkg, func(node ast.Node) bool {
		if node == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		r.parents[node] = stack[len(stack)-1]
		stack = append(stack, node)
		return true
	})
}

func findFunc(file *ast.File, name string) *ast.FuncDecl {
	for _, decl := range file.Decls {
		fd, _ := decl.(*ast.FuncDecl)
		if fd != nil && fd.Name.Name == name {
			return fd
		}
	}
	return nil
}

func delFunc(file *ast.File, name string) *ast.FuncDecl {
	for i, decl := range file.Decls {
		fd, _ := decl.(*ast.FuncDecl)
		if fd != nil && fd.Name.Name == name {
			file.Decls = append(file.Decls[:i], file.Decls[i+1:]...)
			return fd
		}
	}
	return nil
}

func (r *reducer) buildAndRun() []byte {
	cmd := exec.Command("go", r.goArgs...)
	if out, err := cmd.CombinedOutput(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return out
		}
		panic("could not call go build: " + err.Error())
	}
	if !r.toRun {
		return nil
	}
	if out, err := exec.Command(r.outBin).CombinedOutput(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return out
		}
		panic("could not call binary: " + err.Error())
	}
	return nil
}
