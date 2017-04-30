package crasher

func F() {
	println("foo")
	switch nil.(type) {
	}
}
