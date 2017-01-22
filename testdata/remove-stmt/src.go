package crasher

func Crasher() {
	a := []int{1, 2, 3}
	if true {
		a = append(a, 4)
	}
	a[1] = -2
	println(a[10])
}
