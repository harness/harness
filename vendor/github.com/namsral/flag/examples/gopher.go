package main

import (
	"github.com/drone/drone/Godeps/_workspace/src/github.com/namsral/flag"
	"fmt"
)

func main() {
	var (
		config string
		length float64
		age    int
		name   string
		female bool
	)

	flag.StringVar(&config, "config", "", "help message")
	flag.StringVar(&name, "name", "", "help message")
	flag.IntVar(&age, "age", 0, "help message")
	flag.Float64Var(&length, "length", 0, "help message")
	flag.BoolVar(&female, "female", false, "help message")

	flag.Parse()

	fmt.Println("length:", length)
	fmt.Println("age:", age)
	fmt.Println("name:", name)
	fmt.Println("female:", female)
}
