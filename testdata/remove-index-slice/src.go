package crasher

func Crasher() {
	var a *[][]int
	println((*a)[1:][0])
}
