package crasher

func Crasher() {
	a := false
	b := 0
	if a {
		_ = b
	} else {
		panic("panic message")
	}
}
