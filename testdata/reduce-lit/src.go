package crasher

var a = "foo"

func Crasher() {
	_ = a[5]
}

// inlining a would fail, make test faster
var Sink = a
