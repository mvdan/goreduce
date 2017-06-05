package crasher

func Crasher() {
	println([][]int{}[0][1:])
}
