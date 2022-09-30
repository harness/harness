// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package account

import (
	"context"
	"time"

	"github.com/harness/gitness/cli/session"
	"github.com/harness/gitness/cli/util"
	"github.com/harness/gitness/client"
	"gopkg.in/alecthomas/kingpin.v2"
)

type registerCommand struct {
	server string
}

func (c *registerCommand) run(*kingpin.ParseContext) error {
	username, name, email, password := util.Registration()
	httpClient := client.New(c.server)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	ts, err := httpClient.Register(ctx, username, name, email, password)
	if err != nil {
		return err
	}

	return util.StoreSession(&session.Session{
		URI:         c.server,
		ExpiresAt:   ts.Token.ExpiresAt,
		AccessToken: ts.AccessToken,
	})
}

// helper function to register the register command.
func RegisterRegister(app *kingpin.Application) {
	c := new(registerCommand)

	cmd := app.Command("register", "register a user").
		Action(c.run)

	cmd.Arg("server", "server address").
		Default("http://localhost:3000").
		StringVar(&c.server)
}
