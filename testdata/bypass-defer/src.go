package crasher

func Crasher() {
	defer panic("foocrash")
}
