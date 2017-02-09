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
	verbose  = flag.Bool("v", false, "log applied changes to stderr")

	buildFlags = []string{"-ldflags", "-w -s"}
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr,
			"Usage: goreduce -match=re dir fn [build flags]\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Flags to pass to 'go build' can be given as extra arguments. The defaults are:

  -ldflags "-w -s"

Examples:

  goreduce -match='nil pointer' . Crasher
  goreduce -match='index out of range' foo/bar crasher -gcflags=-l
`)
	}
	flag.Parse()
}

func main() {
	args := flag.Args()
	if len(args) < 2 || *matchStr == "" {
		flag.Usage()
		os.Exit(2)
	}
	if err := reduce(args[0], args[1], *matchStr, args[2:]...); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
