// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var dirsGlob = filepath.Join("testdata", "*")

func TestReductions(t *testing.T) {
	dirs, err := filepath.Glob(dirsGlob)
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range dirs {
		name := filepath.Base(dir)
		t.Run(name, testReduction(dir))
	}
}

func readFile(t *testing.T, path string) string {
	bs, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(bs)
}

func testReduction(dir string) func(*testing.T) {
	return func(t *testing.T) {
		// TODO: don't chdir to allow parallel test execution
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}
		orig := []byte(readFile(t, "src.go"))
		defer ioutil.WriteFile("src.go", orig, 0644)
		want := readFile(t, "src.go.min")
		match := strings.TrimRight(readFile(t, "match"), "\n")
		if err := reduce(".", "Crasher", match); err != nil {
			t.Fatal(err)
		}
		got := readFile(t, "src.go")
		if want != got {
			t.Fatalf("unexpected output!\nwant:\n%s\ngot:\n%s\n",
				want, got)
		}
	}
}
