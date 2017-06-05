package crasher

func Crasher() {
	if false {
	} else {
		panic("panic message")
	}
}
