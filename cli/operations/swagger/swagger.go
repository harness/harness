// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package swagger

import (
	"os"

	"github.com/harness/gitness/app/api/openapi"

	"gopkg.in/alecthomas/kingpin.v2"
)

type command struct {
	openAPIService openapi.Service
	path           string
}

func (c *command) run(*kingpin.ParseContext) error {
	spec := c.openAPIService.Generate()
	data, _ := spec.MarshalYAML()
	if c.path == "" {
		os.Stdout.Write(data)
		return nil
	}

	return os.WriteFile(c.path, data, 0o600)
}

// helper function to register the swagger command.
func Register(app *kingpin.Application, openAPIService openapi.Service) {
	c := &command{
		openAPIService: openAPIService,
	}

	cmd := app.Command("swagger", "generate swagger file").
		Hidden().
		Action(c.run)

	cmd.Arg("path", "path to save swagger file").
		StringVar(&c.path)
}
