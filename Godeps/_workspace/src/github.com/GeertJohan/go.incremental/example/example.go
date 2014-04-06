package main

import (
	"fmt"
	"github.com/GeertJohan/go.incremental"
	"runtime"
)

func main() {
	// use max cpu's
	runtime.GOMAXPROCS(runtime.NumCPU())

	// create new incremental.Int
	i := &incremental.Int{}

	// print some numbers
	fmt.Println(i.Next()) // print 1
	fmt.Println(i.Next()) // print 2
	fmt.Println(i.Next()) // print 3

	// create chan to check if goroutines are done
	done := make(chan int)

	// spawn 4 goroutines
	for a := 0; a < 4; a++ {
		// call goroutine with it's number (0-3)
		go func(aa int) {
			// print 10 incremental numbers
			for b := 0; b < 10; b++ {
				fmt.Printf("routine %d: %d\n", aa, i.Next())
			}
			// signal done
			done <- aa
		}(a)
	}

	// wait until all goroutines are done
	for a := 0; a < 4; a++ {
		fmt.Printf("goroutine %d done\n", <-done)
	}
	fmt.Println("all done")
}
