package crasher

var c = 0

func Crasher() {
	a := ""
	_ = a[c]
}
