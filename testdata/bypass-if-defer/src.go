package crasher

func Crasher() string {
	msg := "wrong message"
	if true {
		msg := "panic message"
		defer panic(msg)
	}
	return msg
}
