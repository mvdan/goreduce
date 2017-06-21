package main

func main() {
	msg := 0
	{
		msg := 0.0
		panic(msg)
	}
	panic(msg)
}
