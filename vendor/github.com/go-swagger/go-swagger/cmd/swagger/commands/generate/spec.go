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

package generate

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/go-swagger/go-swagger/scan"
	"github.com/go-swagger/go-swagger/spec"
	"github.com/jessevdk/go-flags"
)

// SpecFile command to generate a swagger spec from a go application
type SpecFile struct {
	BasePath   string         `long:"base-path" short:"b" description:"the base path to use" default:"."`
	ScanModels bool           `long:"scan-models" short:"m" description:"includes models that were annotated with 'swagger:model'"`
	Output     flags.Filename `long:"output" short:"o" description:"the file to write to"`
	Input      flags.Filename `long:"input" short:"i" description:"the file to use as input"`
}

// Execute runs this command
func (s *SpecFile) Execute(args []string) error {
	input, err := loadSpec(string(s.Input))
	if err != nil {
		return err
	}

	var opts scan.Opts
	opts.BasePath = s.BasePath
	opts.Input = input
	opts.ScanModels = s.ScanModels
	swspec, err := scan.Application(opts)
	if err != nil {
		return err
	}

	return writeToFile(swspec, string(s.Output))
}

var (
	newLine = []byte("\n")
)

func loadSpec(input string) (*spec.Swagger, error) {
	if fi, err := os.Stat(input); err == nil {
		if fi.IsDir() {
			return nil, fmt.Errorf("expected %q to be a file not a directory", input)
		}
		sp, err := spec.Load(input)
		if err != nil {
			return nil, err
		}
		return sp.Spec(), nil
	}
	return nil, nil
}

func writeToFile(swspec *spec.Swagger, output string) error {
	b, err := json.Marshal(swspec)
	if err != nil {
		return err
	}
	if output == "" {
		fmt.Println(string(b))
		return nil
	}
	return ioutil.WriteFile(output, b, 0644)
}
