package main

var a = "foo"

func crasher() {
	const b = 1
	println(a[(+b + c)])
}

func main() {
}
