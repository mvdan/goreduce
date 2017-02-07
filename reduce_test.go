// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
)

var dirsGlob = filepath.Join("testdata", "*")

func TestReductions(t *testing.T) {
	paths, err := filepath.Glob(dirsGlob)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		name := filepath.Base(path)
		t.Run(name, testReduction(name))
	}
}

func readFile(t testing.TB, dir, path string) string {
	bs, err := ioutil.ReadFile(filepath.Join(dir, path))
	if err != nil {
		t.Fatal(err)
	}
	return string(bs)
}

func testReduction(name string) func(*testing.T) {
	return func(t *testing.T) {
		t.Parallel()
		dir := filepath.Join("testdata", name)
		orig := []byte(readFile(t, dir, "src.go"))
		defer ioutil.WriteFile(filepath.Join(dir, "src.go"), orig, 0644)
		want := readFile(t, dir, "src.go.min")
		match := strings.TrimRight(readFile(t, dir, "match"), "\n")
		impPath := "./testdata/" + name
		if err := reduce(impPath, "Crasher", match); err != nil {
			t.Fatal(err)
		}
		got := readFile(t, dir, "src.go")
		if want != got {
			t.Fatalf("unexpected output!\nwant:\n%s\ngot:\n%s\n",
				want, got)
		}
	}
}

func BenchmarkReduce(b *testing.B) {
	impPath := "./testdata/remove-stmt"
	dir := filepath.Join("testdata", "remove-stmt")
	orig := []byte(readFile(b, dir, "src.go"))
	match := strings.TrimRight(readFile(b, dir, "match"), "\n")
	for i := 0; i < b.N; i++ {
		if err := reduce(impPath, "Crasher", match); err != nil {
			b.Fatal(err)
		}
		if err := ioutil.WriteFile(filepath.Join(dir, "src.go"), orig, 0644); err != nil {
			b.Fatal(err)
		}
	}
}
