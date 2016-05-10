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
	"errors"

	"github.com/go-swagger/go-swagger/generator"
)

// Operation the generate operation files command
type Operation struct {
	shared
	Name          []string `long:"name" short:"n" required:"true" description:"the operations to generate, repeat for multiple"`
	Tags          []string `long:"tags" description:"the tags to include, if not specified defaults to all"`
	Principal     string   `short:"P" long:"principal" description:"the model to use for the security principal"`
	DefaultScheme string   `long:"default-scheme" description:"the default scheme for this API" default:"http"`
	NoHandler     bool     `long:"skip-handler" description:"when present will not generate an operation handler"`
	NoStruct      bool     `long:"skip-parameters" description:"when present will not generate the parameter model struct"`
	NoResponses   bool     `long:"skip-responses" description:"when present will not generate the response model struct"`
	DumpData      bool     `long:"dump-data" description:"when present dumps the json for the template generator instead of generating files"`
}

// Execute generates a model file
func (o *Operation) Execute(args []string) error {
	if o.DumpData && len(o.Name) > 1 {
		return errors.New("only 1 operation at a time is supported for dumping data")
	}
	return generator.GenerateServerOperation(
		o.Name,
		o.Tags,
		!o.NoHandler,
		!o.NoStruct,
		!o.NoResponses,
		generator.GenOpts{
			Spec:          string(o.Spec),
			Target:        string(o.Target),
			APIPackage:    o.APIPackage,
			ModelPackage:  o.ModelPackage,
			ServerPackage: o.ServerPackage,
			ClientPackage: o.ClientPackage,
			Principal:     o.Principal,
			DumpData:      o.DumpData,
			DefaultScheme: o.DefaultScheme,
			TemplateDir:   string(o.TemplateDir),
		})
}
