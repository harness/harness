// Copyright 2015 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"log"
	"os"

	"github.com/go-swagger/go-swagger/cmd/swagger/commands"
	"github.com/jessevdk/go-flags"
)

var opts struct {
	// Version bool `long:"version" short:"v" description:"print the version of the command"`
}

func main() {
	parser := flags.NewParser(&opts, flags.Default)
	parser.ShortDescription = "helps you keep your API well described"
	parser.LongDescription = `
Swagger tries to support you as best as possible when building API's.

It aims to represent the contract of your API with a language agnostic description of your application in json or yaml.
`
	_, err := parser.AddCommand("validate", "validate the swagger document", "validate the provided swagger document against a swagger spec", &commands.ValidateSpec{})
	if err != nil {
		log.Fatal(err)
	}

	_, err = parser.AddCommand("init", "initialize a spec document", "initialize a swagger spec document", &commands.InitCmd{})
	if err != nil {
		log.Fatal(err)
	}

	_, err = parser.AddCommand("version", "print the version", "print the version of the swagger command", &commands.PrintVersion{})
	if err != nil {
		log.Fatal(err)
	}

	genpar, err := parser.AddCommand("generate", "genererate go code", "generate go code for the swagger spec file", &commands.Generate{})
	if err != nil {
		log.Fatalln(err)
	}
	for _, cmd := range genpar.Commands() {
		switch cmd.Name {
		case "spec":
			cmd.ShortDescription = "generate a swagger spec document from a go application"
			cmd.LongDescription = cmd.ShortDescription
		case "client":
			cmd.ShortDescription = "generate all the files for a client library"
			cmd.LongDescription = cmd.ShortDescription
		case "server":
			cmd.ShortDescription = "generate all the files for a server application"
			cmd.LongDescription = cmd.ShortDescription
		case "model":
			cmd.ShortDescription = "generate one or more models from the swagger spec"
			cmd.LongDescription = cmd.ShortDescription
		case "support":
			cmd.ShortDescription = "generate supporting files like the main function and the api builder"
			cmd.LongDescription = cmd.ShortDescription
		case "operation":
			cmd.ShortDescription = "generate one or more server operations from the swagger spec"
			cmd.LongDescription = cmd.ShortDescription
		}
	}

	if _, err := parser.Parse(); err != nil {
		os.Exit(1)
	}
}
