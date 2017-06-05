package crasher

func Crasher() {
	fn := func() {
		panic(0)
	}
	fn()
}
