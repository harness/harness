// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package account

import (
	"context"
	"time"

	"github.com/harness/gitness/cli/textui"

	"github.com/harness/gitness/cli/session"
	"github.com/harness/gitness/client"
	"gopkg.in/alecthomas/kingpin.v2"
)

type Session interface {
	SetURI(uri string) session.Session
	SetExpiresAt(expiresAt int64) session.Session
	SetAccessToken(token string) session.Session
	Path() string
	Store() error
}

type registerCommand struct {
	client  client.Client
	session Session
	server  string
}

func (c *registerCommand) run(*kingpin.ParseContext) error {
	username, name, email, password := textui.Registration()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	ts, err := c.client.Register(ctx, username, name, email, password)
	if err != nil {
		return err
	}

	return c.session.SetURI(c.server).SetExpiresAt(ts.Token.ExpiresAt).SetAccessToken(ts.AccessToken).Store()
}

// RegisterRegister helper function to register the register command.
func RegisterRegister(app *kingpin.Application, client client.Client, session *session.Session) {
	c := &registerCommand{
		client:  client,
		session: session,
	}

	cmd := app.Command("register", "register a user").
		Action(c.run)

	cmd.Arg("server", "server address").
		Default(session.URI).
		StringVar(&c.server)
}
