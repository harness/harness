// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package users

import (
	"context"
	"time"

	"github.com/harness/gitness/client"

	"gopkg.in/alecthomas/kingpin.v2"
)

type deleteCommand struct {
	client client.Client
	email  string
}

func (c *deleteCommand) run(*kingpin.ParseContext) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	return c.client.UserDelete(ctx, c.email)
}

// helper function registers the user delete command.
func registerDelete(app *kingpin.CmdClause, client client.Client) {
	c := &deleteCommand{
		client: client,
	}

	cmd := app.Command("delete", "delete a user").
		Action(c.run)

	cmd.Arg("id or email", "user id or email").
		Required().
		StringVar(&c.email)
}
