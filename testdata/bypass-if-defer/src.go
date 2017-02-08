package crasher

func Crasher() {
	if true {
		defer panic("panic message")
	}
}
