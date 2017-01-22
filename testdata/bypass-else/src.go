package crasher

func Crasher() {
	a := []int{0}
	if false {
	} else {
		a = nil
	}

	println(a[0])
}
