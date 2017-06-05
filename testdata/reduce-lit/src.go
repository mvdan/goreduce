package main

var a = "foo"

func main() {
	_ = a[5]
}

// inlining a would fail, make test faster
var _ = a
