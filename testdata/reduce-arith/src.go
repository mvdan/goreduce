package crasher

func Crasher() {
	a := ""
	println(a[2+(-1)])
}
