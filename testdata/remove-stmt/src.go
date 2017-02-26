package crasher

import (
	"unsafe"
	foo "errors"
)

// Crasher just crashes.
func Crasher() {
	b := false
	bs := []bool{b}
	var p unsafe.Pointer
	_, _ = foo.New, foo.New("")
	println(p)
	_ = false || bs[12345678987654321]
}
