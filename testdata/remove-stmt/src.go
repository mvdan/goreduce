package crasher

import (
	"os"
	foo "sync"
)

func Crasher() {
	var a []int
	_ = foo.Once{}
	_ = os.File{}
	println(a[0])
}
