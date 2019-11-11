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
	shellStr = flag.String("run", "", "shell command to test reductions")
	verbose  = flag.Bool("v", false, "log applied changes to stderr")

	shellStrBuild = `go build -ldflags "-w -s"`
	shellStrRun   = `go build -ldflags "-w -s" -o out && ./out`
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr,
			"Usage: goreduce -match=re [-run=cmd] dir\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
If -run=cmd is omitted, the default for non-main packages is:

  `+shellStrBuild+`

And for main packages:

  `+shellStrRun+`

The shell code is run in a Bash-compatible shell interpreter. The
package being reduced will be in its current directory.

To catch a run-time error/crash entering main:

  goreduce -match 'index out of range' .

To catch a build error/crash with custom build flags:

  goreduce -match 'internal compiler error' . 'go build -gcflags "-c=2"'

Note that you may also call a script or any other program.
`)
	}
}

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 || *matchStr == "" {
		flag.Usage()
		os.Exit(2)
	}
	if err := reduce(args[0], *matchStr, os.Stderr, *shellStr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
