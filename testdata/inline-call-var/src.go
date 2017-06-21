package main

func main() {
	fn := func() {
		panic(0)
	}
	fn()
}
