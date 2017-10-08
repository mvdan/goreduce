package main

type foo int

func (f foo) crash() {
	panic(0)
}

func main() {
	var f foo
	f.crash()
}
