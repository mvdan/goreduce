package crasher

func Crasher() {
	switch {
	case false:
	case true:
		panic(0)
	}
}
