package main

import (
	"flag"
	"fmt"
	amber "github.com/eknkc/amber"
	"os"
)

var prettyPrint bool
var lineNumbers bool

func init() {
	flag.BoolVar(&prettyPrint, "prettyprint", true, "Use pretty indentation in output html.")
	flag.BoolVar(&prettyPrint, "pp", true, "Use pretty indentation in output html.")

	flag.BoolVar(&lineNumbers, "linenos", true, "Enable debugging information in output html.")
	flag.BoolVar(&lineNumbers, "ln", true, "Enable debugging information in output html.")

	flag.Parse()
}

func main() {
	input := flag.Arg(0)

	if len(input) == 0 {
		fmt.Fprintln(os.Stderr, "Please provide an input file. (amberc input.amber)")
		os.Exit(1)
	}

	cmp := amber.New()
	cmp.PrettyPrint = prettyPrint
	cmp.LineNumbers = lineNumbers

	err := cmp.ParseFile(input)

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	err = cmp.CompileWriter(os.Stdout)

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
