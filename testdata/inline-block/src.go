package crasher

func Crasher() int {
	msg := 1
	{
		msg := 0
		panic(msg)
	}
	return msg
}
