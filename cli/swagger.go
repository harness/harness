// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package cli

import (
	"io/ioutil"
	"os"

	"github.com/harness/scm/internal/api/openapi"
	"gopkg.in/alecthomas/kingpin.v2"
)

type swaggerCommand struct {
	path string
}

func (c *swaggerCommand) run(*kingpin.ParseContext) error {
	spec := openapi.Generate()
	data, _ := spec.MarshalYAML()
	if c.path == "" {
		os.Stdout.Write(data)
		return nil
	}
	return ioutil.WriteFile(c.path, data, 0600)
}

// helper function to register the swagger command.
func registerSwagger(app *kingpin.Application) {
	c := new(swaggerCommand)

	cmd := app.Command("swagger", "generate swagger file").
		Hidden().
		Action(c.run)

	cmd.Arg("path", "path to save swagger file").
		StringVar(&c.path)
}
