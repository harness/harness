// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package execution

import (
	"github.com/harness/scm/cli/util"

	"gopkg.in/alecthomas/kingpin.v2"
)

type deleteCommand struct {
	pipeline string
	slug     string
}

func (c *deleteCommand) run(*kingpin.ParseContext) error {
	client, err := util.Client()
	if err != nil {
		return err
	}
	return client.ExecutionDelete(c.pipeline, c.slug)
}

// helper function registers the user delete command
func registerDelete(app *kingpin.CmdClause) {
	c := new(deleteCommand)

	cmd := app.Command("delete", "delete a execution").
		Action(c.run)

	cmd.Arg("pipeline ", "pipeline slug").
		Required().
		StringVar(&c.pipeline)

	cmd.Arg("slug ", "execution slug").
		Required().
		StringVar(&c.slug)
}
