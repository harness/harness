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

import "github.com/go-swagger/go-swagger/generator"

// Client the command to generate a swagger client
type Client struct {
	shared
	Name            string   `long:"name" short:"A" description:"the name of the application, defaults to a mangled value of info.title"`
	Operations      []string `long:"operation" short:"O" description:"specify an operation to include, repeat for multiple"`
	Tags            []string `long:"tags" description:"the tags to include, if not specified defaults to all"`
	Principal       string   `long:"principal" short:"P" description:"the model to use for the security principal"`
	Models          []string `long:"model" short:"M" description:"specify a model to include, repeat for multiple"`
	DefaultScheme   string   `long:"default-scheme" description:"the default scheme for this client" default:"http"`
	DefaultProduces string   `long:"default-consumes" description:"the default mime type that API operations produce" default:"application/json"`
	SkipModels      bool     `long:"skip-models" description:"no models will be generated when this flag is specified"`
	SkipOperations  bool     `long:"skip-operations" description:"no operations will be generated when this flag is specified"`
}

// Execute runs this command
func (c *Client) Execute(args []string) error {
	opts := generator.GenOpts{
		Spec:              string(c.Spec),
		Target:            string(c.Target),
		APIPackage:        c.APIPackage,
		ModelPackage:      c.ModelPackage,
		ServerPackage:     c.ServerPackage,
		ClientPackage:     c.ClientPackage,
		Principal:         c.Principal,
		DefaultScheme:     c.DefaultScheme,
		DefaultProduces:   c.DefaultProduces,
		IncludeModel:      !c.SkipModels,
		IncludeValidator:  !c.SkipModels,
		IncludeHandler:    !c.SkipOperations,
		IncludeParameters: !c.SkipOperations,
		IncludeResponses:  !c.SkipOperations,
		TemplateDir:       string(c.TemplateDir),
	}
	if err := generator.GenerateClient(c.Name, c.Models, c.Operations, opts); err != nil {
		return err
	}

	return nil
}
