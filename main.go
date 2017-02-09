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

	ldflags = flag.String("ldflags", "-w -s", "as passed to 'go build'")
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: goreduce -match=re pkg func\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr,
			"\nExample: goreduce -match='nil pointer' . Crasher\n")
	}
	flag.Parse()
}

func main() {
	args := flag.Args()
	if len(args) != 2 || *matchStr == "" {
		flag.Usage()
		os.Exit(2)
	}
	if err := reduce(args[0], args[1], *matchStr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
