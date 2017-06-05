package crasher

func Crasher() {
	defer panic(0)
}
