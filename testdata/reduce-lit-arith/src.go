package crasher

var a = "foo"

func Crasher() {
	println(a[(+1+2)])
}
