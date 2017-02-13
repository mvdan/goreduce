package crasher

func Crasher() {
	a := true
	if a {
		defer panic("panic message")
	}
}
