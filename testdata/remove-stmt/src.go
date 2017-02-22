package crasher

import (
	"os"
	foo "sync"
)

func Crasher() {
	a := []bool{false}
	var b *os.File
	_, _ = foo.Once{}, foo.Once{}
	println(b)
	_ = false || a[1]
}
