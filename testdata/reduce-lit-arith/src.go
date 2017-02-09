package main

var a = "foo"

func crasher() {
	println(a[(+1 + b)])
}

func main() {
}
