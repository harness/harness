package main

import (
	"fmt"
	goflags "github.com/jessevdk/go-flags" // rename import to `goflags` (file scope) so we can use `var flags` (package scope)
	"go/build"
	"os"
)

// flags
var flags struct {
	Verbose    bool   `long:"verbose" short:"v" description:"Show verbose debug information"`
	ImportPath string `long:"import-path" short:"i" description:"Import path to use. Using PWD when left empty."`

	Append struct {
		Executable string `long:"exec" description:"Executable to append" required:"true"`
	} `command:"append"`

	EmbedGo   struct{} `command:"embed-go" alias:"embed"`
	EmbedSyso struct{} `command:"embed-syso"`
	Clean     struct{} `command:"clean"`
}

// flags parser
var flagsParser *goflags.Parser

// initFlags parses the given flags.
// when the user asks for help (-h or --help): the application exists with status 0
// when unexpected flags is given: the application exits with status 1
func parseArguments() {
	// create flags parser in global var, for flagsParser.Active.Name (operation)
	flagsParser = goflags.NewParser(&flags, goflags.Default)

	// parse flags
	args, err := flagsParser.Parse()
	if err != nil {
		// assert the err to be a flags.Error
		flagError := err.(*goflags.Error)
		if flagError.Type == goflags.ErrHelp {
			// user asked for help on flags.
			// program can exit successfully
			os.Exit(0)
		}
		if flagError.Type == goflags.ErrUnknownFlag {
			fmt.Println("Use --help to view available options.")
			os.Exit(1)
		}
		if flagError.Type == goflags.ErrRequired {
			os.Exit(1)
		}
		fmt.Printf("Error parsing flags: %s\n", err)
		os.Exit(1)
	}

	// error on left-over arguments
	if len(args) > 0 {
		fmt.Printf("Unexpected arguments: %s\nUse --help to view available options.", args)
		os.Exit(1)
	}

	// default ImportPath to pwd when not set
	if len(flags.ImportPath) == 0 {
		pwd, err := os.Getwd()
		if err != nil {
			fmt.Printf("error getting pwd: %s\n", err)
			os.Exit(1)
		}
		verbosef("using pwd as import path\n")
		// find non-absolute path for this pwd
		pkg, err := build.ImportDir(pwd, build.FindOnly)
		if err != nil {
			fmt.Printf("error using current directory as import path: %s\n", err)
			os.Exit(1)
		}
		flags.ImportPath = pkg.ImportPath
		verbosef("using import path: %s\n", flags.ImportPath)
		return
	}
}
