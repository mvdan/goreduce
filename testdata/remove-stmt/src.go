package crasher

import (
	"os"
	foo "sync"
)

func Crasher() {
	var a []int
	var b *os.File
	_, _ = foo.Once{}, foo.Once{}
	println(b)
	println(a[0])
}
