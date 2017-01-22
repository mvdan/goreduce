// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import (
	"flag"
	"fmt"
	"os"
)

var matchStr = flag.String("match", "", "")

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "func name missing: goreduce funcName")
		os.Exit(1)
	}
	if err := reduce(args[0], *matchStr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
