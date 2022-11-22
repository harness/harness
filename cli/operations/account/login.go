// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package account

import (
	"context"
	"time"

	"github.com/harness/gitness/cli/session"
	"github.com/harness/gitness/cli/textui"
	"github.com/harness/gitness/client"

	"gopkg.in/alecthomas/kingpin.v2"
)

type loginCommand struct {
	client  client.Client
	session Session
	server  string
}

func (c *loginCommand) run(*kingpin.ParseContext) error {
	username, password := textui.Credentials()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	ts, err := c.client.Login(ctx, username, password)
	if err != nil {
		return err
	}

	return c.session.SetURI(c.server).SetExpiresAt(ts.Token.ExpiresAt).SetAccessToken(ts.AccessToken).Store()
}

// RegisterLogin helper function to register the logout command.
func RegisterLogin(app *kingpin.Application, client client.Client, session *session.Session) {
	c := &loginCommand{
		client:  client,
		session: session,
	}

	cmd := app.Command("login", "login to the remote server").
		Action(c.run)

	cmd.Arg("server", "server address").
		Default(session.URI).
		StringVar(&c.server)
}
