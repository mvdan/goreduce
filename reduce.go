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

	outBin  string
	goArgs  []string
	dstFile *os.File
	dstBuf  *bytes.Buffer

	tries     int
	didChange bool

	tried map[string]bool

	walker
}

var errNoReduction = fmt.Errorf("could not reduce program")

func reduce(dir, funcName, match string, logOut io.Writer, bflags ...string) error {
	r := &reducer{
		dir:    dir,
		logOut: logOut,
		tried:  make(map[string]bool),
		dstBuf: bytes.NewBuffer(nil),
	}
	tdir, err := ioutil.TempDir("", "goreduce")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tdir)
	r.tconf.Importer = importer.Default()
	r.tconf.Error = func(err error) {
		if terr, ok := err.(types.Error); ok && terr.Soft {
			// don't stop type-checking on soft errors
			return
		}
		panic("types.Check should not error here: " + err.Error())
	}
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
	var origMain *ast.FuncDecl
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
		if funcName != "main" {
			if fd := delFunc(file, "main"); fd != nil && file == r.file {
				origMain = fd
			}
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
	if funcName != "" && !foundFunc {
		return fmt.Errorf("top-level func %s does not exist", funcName)
	}
	mfname := filepath.Join(tdir, mainFile)
	if funcName != "main" {
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
		fmt.Fprintf(r.logOut, "%s:%d: %s (%d tries)\n",
			pos.Filename, pos.Line, fmt.Sprintf(format, a...), r.tries)
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

func (r *reducer) okChange() bool {
	if r.didChange {
		return false
	}
	r.tries++
	r.dstBuf.Reset()
	if err := rawPrinter.Fprint(r.dstBuf, r.fset, r.file); err != nil {
		return false
	}
	newSrc := r.dstBuf.String()
	if r.tried[newSrc] {
		return false
	}
	r.tried[newSrc] = true
	if err := r.dstFile.Truncate(0); err != nil {
		return false
	}
	if _, err := r.dstFile.Seek(0, io.SeekStart); err != nil {
		return false
	}
	if _, err := r.dstFile.Write(r.dstBuf.Bytes()); err != nil {
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
		// Update type info after the AST changes
		r.tconf.Check(r.dir, r.fset, r.files, r.info)
		r.fillUses()

		r.didChange = false
		r.walk(r.file, r.reduceNode)
		if !r.didChange {
			if *verbose {
				fmt.Fprintf(r.logOut, "gave up after %d final tries\n", r.tries)
			}
			return
		}
		anyChanges = true
	}
}

func (r *reducer) fillUses() {
	r.useIdents = make(map[types.Object][]*ast.Ident)
	for id, obj := range r.info.Uses {
		if pkg := obj.Pkg(); pkg == nil || pkg.Name() != "main" {
			// builtin or declared outside of our pkg
			continue
		}
		r.useIdents[obj] = append(r.useIdents[obj], id)
	}
}

func findFunc(file *ast.File, name string) *ast.FuncDecl {
	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if ok && fd.Name.Name == name {
			return fd
		}
	}
	return nil
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

func (r *reducer) buildAndRun() []byte {
	cmd := exec.Command("go", r.goArgs...)
	if out, err := cmd.CombinedOutput(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return out
		}
		panic("could not call go build: " + err.Error())
	}
	if out, err := exec.Command(r.outBin).CombinedOutput(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return out
		}
		panic("could not call binary: " + err.Error())
	}
	return nil
}
