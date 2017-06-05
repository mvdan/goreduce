package crasher

func Crasher() {
	func() {
		panic(0)
	}()
}
