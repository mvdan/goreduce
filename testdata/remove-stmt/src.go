package crasher

import (
	"os"
	foo "sync"
)

func Crasher() {
	var a []int
	_, _ = foo.Once{}, foo.Once{}
	_ = os.File{}
	println(a[0])
}
