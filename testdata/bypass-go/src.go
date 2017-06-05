package crasher

func Crasher() {
	go panic(0)
	select {}
}
