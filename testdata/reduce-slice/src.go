package crasher

func Crasher() {
	_ = []int{}[1:2]
}
