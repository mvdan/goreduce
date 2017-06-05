package crasher

var msg = "panic message"

func Crasher() {
	panic(msg)
}
