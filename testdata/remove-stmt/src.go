package crasher

func Crasher() {
	a := []int{}
	if true {
		a = append(a, 3)
	}
	a[1] = -2
	println(a[0])
}
