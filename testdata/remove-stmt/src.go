package crasher

// Crasher just crashes.
func Crasher() {
	println("foo")
	panic(0)
}
