// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package account

import (
	"context"
	"time"

	"github.com/harness/gitness/cli/provide"
	"github.com/harness/gitness/cli/textui"
	"github.com/harness/gitness/internal/api/controller/user"

	"gopkg.in/alecthomas/kingpin.v2"
)

type loginCommand struct {
	server string
}

func (c *loginCommand) run(*kingpin.ParseContext) error {
	ss := provide.NewSession()

	loginIdentifier, password := textui.Credentials()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	in := &user.LoginInput{
		LoginIdentifier: loginIdentifier,
		Password:        password,
	}

	ts, err := provide.OpenClient(c.server).Login(ctx, in)
	if err != nil {
		return err
	}

	return ss.
		SetURI(c.server).
		// login token always has an expiry date
		SetExpiresAt(*ts.Token.ExpiresAt).
		SetAccessToken(ts.AccessToken).
		Store()
}

// RegisterLogin helper function to register the logout command.
func RegisterLogin(app *kingpin.Application) {
	c := &loginCommand{}

	cmd := app.Command("login", "login to the remote server").
		Action(c.run)

	cmd.Arg("server", "server address").
		Default(provide.DefaultServerURI).
		StringVar(&c.server)
}
