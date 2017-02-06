package crasher

func Crasher() {
	go panic("panic message")
	var c chan int
	<-c
}
