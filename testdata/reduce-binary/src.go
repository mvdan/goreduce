package crasher

// Crasher just crashes.
func Crasher() {
	_ = false || *(*bool)(nil)
}
