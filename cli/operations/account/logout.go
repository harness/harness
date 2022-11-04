// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package account

import (
	"os"

	"github.com/harness/gitness/cli/session"

	"gopkg.in/alecthomas/kingpin.v2"
)

type logoutCommand struct {
	session Session
}

func (c *logoutCommand) run(*kingpin.ParseContext) error {
	return os.Remove(c.session.Path())
}

// RegisterLogout helper function to register the logout command.
func RegisterLogout(app *kingpin.Application, s *session.Session) {
	c := &logoutCommand{
		session: s,
	}

	app.Command("logout", "logout from the remote server").
		Action(c.run)
}
