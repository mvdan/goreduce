package crasher

import (
	"os"
	foo "sync"
)

func Crasher() {
	b := false
	bs := []bool{b}
	var f *os.File
	_, _ = foo.Once{}, foo.Once{}
	println(f)
	_ = false || bs[12345678987654321]
}
