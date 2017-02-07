package crasher

import (
	foo "sync"
)

func Crasher() {
	var a []int
	_ = foo.Once{}
	println(a[0])
}
