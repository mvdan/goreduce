package crasher

// Crasher just crashes.
func Crasher() {
	var b = false
	bs := []bool{b}
	_ = false || bs[12345678987654321]
}
