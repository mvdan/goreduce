// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var write = flag.Bool("w", false, "write test outputs")

func TestReductions(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("testdata", "*"))
	if err != nil {
		t.Fatal(err)
	}
	*verbose = true
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

func writeFile(t testing.TB, dir, path, cont string) {
	err := ioutil.WriteFile(filepath.Join(dir, path), []byte(cont), 0644)
	if err != nil {
		t.Fatal(err)
	}
}

func testReduction(name string) func(*testing.T) {
	return func(t *testing.T) {
		t.Parallel()
		dir := filepath.Join("testdata", name)
		orig := []byte(readFile(t, dir, "src.go"))
		defer ioutil.WriteFile(filepath.Join(dir, "src.go"), orig, 0644)
		want := readFile(t, dir, "src.go.min")
		match := strings.TrimRight(readFile(t, dir, "match"), "\n")
		call := strings.TrimRight(readFile(t, dir, "call"), "\n")
		impPath := "./testdata/" + name
		var buf bytes.Buffer
		if err := reduce(impPath, call, match, &buf); err != nil {
			t.Fatal(err)
		}
		got := readFile(t, dir, "src.go")
		if want != got {
			t.Fatalf("unexpected program output\nwant:\n%sgot:\n%s",
				want, got)
		}
		// remove testdata/<dir>/ bit
		rawLog := buf.String()
		buf.Reset()
		for _, line := range strings.Split(rawLog, "\n") {
			if line == "" {
				break
			}
			line = strings.TrimPrefix(line, dir+string(filepath.Separator))
			buf.WriteString(line)
			buf.WriteByte('\n')
		}
		gotLog := buf.String()
		wantLog := readFile(t, dir, "log")
		if wantLog != gotLog {
			if *write {
				writeFile(t, dir, "log", gotLog)
			} else {
				t.Fatalf("unexpected log output\nwant:\n%sgot:\n%s",
					wantLog, gotLog)
			}
		}
	}
}

func BenchmarkReduce(b *testing.B) {
	dir, err := ioutil.TempDir("", "goreduce")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(dir)
	if err := os.Chdir(dir); err != nil {
		b.Fatal(err)
	}
	orig := []byte(`package crasher

import "sync"

func Crasher() {
	var a []int
	_ = sync.Once{}
	println(a[0])
}`)
	for i := 0; i < b.N; i++ {
		if err := ioutil.WriteFile("src.go", orig, 0644); err != nil {
			b.Fatal(err)
		}
		err := reduce(".", "Crasher", "index out of range", ioutil.Discard)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestReduceErrs(t *testing.T) {
	t.Parallel()
	tests := [...]struct {
		dir, funcName, match string
		errCont              string
	}{
		{"missing-dir", "fn", "[", "missing closing ]"},
		{"missing-dir", "fn", ".", "no such file"},
		{"testdata/remove-stmt", "missing-fn", ".", "top-level func"},
		{"testdata/remove-stmt", "Crasher", "no-match", "does not match"},
	}
	for _, tc := range tests {
		err := reduce(tc.dir, tc.funcName, tc.match, ioutil.Discard)
		if err == nil || !strings.Contains(err.Error(), tc.errCont) {
			t.Fatalf("wanted error conatining %q, got: %v",
				tc.errCont, err)
		}
	}
}
