package main

func crasher() {
	a := ""
	_ = a[(-0 + 0)]
}

func main() {
}
