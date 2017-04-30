package crasher

import (
	"unsafe"
	foo "errors"
)

func Crasher() {
	_, _ = foo.New(""), unsafe.Sizeof(0)
	panic("foocrash")
}
