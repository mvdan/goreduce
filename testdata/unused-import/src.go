package main

import (
	foo "errors"
	"unsafe"
)

func main() {
	_, _ = foo.New(""), unsafe.Sizeof(0)
	panic(0)
}
