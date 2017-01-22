package crasher

func Crasher() {
	a := []int{0}
	if true {
		a = nil
	}

	println(a[0])
}
