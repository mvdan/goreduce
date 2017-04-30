package crasher

import (
	foo "errors"
	"unsafe"
)

func Crasher() {
	_, _ = foo.New(""), unsafe.Sizeof(0)
	panic("foocrash")
}
