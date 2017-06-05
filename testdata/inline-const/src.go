package crasher

const msg = "panic message"

func Crasher() {
	panic(msg)
}
