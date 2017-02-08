package crasher

func Crasher() {
	a := []*int{}
	println(a[1:2:3])
}
