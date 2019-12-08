package main

import "fmt"
import "time"

func f(from string) {
	for i := 0; i < 5; i++ {
		fmt.Println(from, ":", i)
		time.Sleep(2 * time.Second)
	}
}

func f2(from string, i int) {
	fmt.Println(from, ":", i)
	time.Sleep(2 * time.Second)
}

func main() {

	//		f("direct", i)
	go f("goroutine")

	for i := 0; i < 5; i++ {
		go f2("goroutine2", i)
	}
	go func(msg string) {
		fmt.Println(msg)
	}("going")

	//	fmt.Scanln()
	fmt.Println("done")
}
