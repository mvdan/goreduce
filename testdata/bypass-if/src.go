package crasher

func Crasher() {
	if true {
		panic("panic message")
	}
}
