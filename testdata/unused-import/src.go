package crasher

import "sync"

func Crasher() {
	var a []int
	_ = sync.Once{}
	println(a[0])
}
