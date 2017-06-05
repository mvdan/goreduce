package crasher

func Crasher() {
	fn()
}

func fn() {
	panic(0)
}
