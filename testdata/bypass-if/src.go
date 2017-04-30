package crasher

func Crasher() string {
	msg := "wrong message"
	if true {
		msg := "panic message"
		panic(msg)
	}
	return msg
}
