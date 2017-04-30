package crasher

func Crasher() {
	go panic("panic message")
	select {}
}
