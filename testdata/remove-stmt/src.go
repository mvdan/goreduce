package main

// main just crashes.
func main() {
	var _ = "foo"
	panic(0)
}
