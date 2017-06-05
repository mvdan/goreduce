package crasher

// Crasher just crashes.
func Crasher() {
	var _ = "foo"
	panic(0)
}
