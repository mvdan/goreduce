package main

func main() {
	go panic(0)
	select {}
}
