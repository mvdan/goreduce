package crasher

func Crasher() {
	var a *[]int
	b := *a
	println(b[1])
}
