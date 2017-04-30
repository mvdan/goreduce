package crasher

const c = 0

func Crasher() {
	a := ""
	_ = a[c]
}
