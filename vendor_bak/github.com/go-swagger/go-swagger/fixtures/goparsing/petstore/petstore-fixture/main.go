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

	"github.com/go-swagger/go-swagger/fixtures/goparsing/petstore"
	"github.com/go-swagger/go-swagger/fixtures/goparsing/petstore/rest"
)

var (
	// Version is a compile time constant, injected at build time
	Version string
)

// This is an application that doesn't actually do anything,
// it's used for testing the scanner
func main() {
	// this has no real purpose besides making the import present in this main package.
	// without this line the meta info for the swagger doc wouldn't be discovered
	petstore.APIVersion = Version

	// This servers na hypothetical API
	if err := rest.ServeAPI(); err != nil {
		log.Fatal(err)
	}
}
