package crasher

import (
	"os"
	foo "sync"
)

func Crasher() {
	var a []bool
	var b *os.File
	_, _ = foo.Once{}, foo.Once{}
	println(b)
	_ = false || a[0]
}
