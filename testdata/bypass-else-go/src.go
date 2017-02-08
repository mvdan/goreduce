package crasher

func Crasher() {
	if false {
	} else {
		go panic("panic message")
	}
	var c chan int
	<-c
}
