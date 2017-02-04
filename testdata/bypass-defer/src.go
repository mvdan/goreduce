package crasher

func Crasher() {
	defer panic("panic message")
}
