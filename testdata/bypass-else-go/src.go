package crasher

func Crasher() {
	a := false
	b := 0
	if a {
		_ = b
	} else {
		go panic("panic message")
	}
	var c chan int
	<-c
}
