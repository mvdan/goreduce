// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	matchStr = flag.String("match", "", "regexp to match the output")
	runStr   = flag.String("run", "", "code to run when reducing run-time errors")
	verbose  = flag.Bool("v", false, "log applied changes to stderr")

	buildFlags = []string{"-ldflags", "-w -s"}
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr,
			"Usage: goreduce -match=re [-run=stmt] dir [build flags]\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Flags to pass to 'go build' can be given as extra arguments. The defaults are:

  -ldflags "-w -s"

To catch a run-time error/crash entering main:

  goreduce -match 'index out of range' -run=main .

Like above, but using a custom main body:

  goreduce -match 'index out of range' -run='Crasher("foo")' .

To catch a build error/crash with build flags:

  goreduce -match 'internal compiler error' . -gcflags '-c=2'
`)
	}
	flag.Parse()
}

func main() {
	args := flag.Args()
	if len(args) < 1 || *matchStr == "" {
		flag.Usage()
		os.Exit(2)
	}
	if err := reduce(args[0], *runStr, *matchStr, os.Stderr, args[1:]...); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
