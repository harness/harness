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

package account

import (
	"context"
	"time"

	"github.com/harness/gitness/app/api/controller/user"
	"github.com/harness/gitness/cli/provide"
	"github.com/harness/gitness/cli/session"
	"github.com/harness/gitness/cli/textui"

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
	server string
}

func (c *registerCommand) run(*kingpin.ParseContext) error {
	ss := provide.NewSession()

	uid, displayName, email, password := textui.Registration()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	input := &user.RegisterInput{
		UID:         uid,
		Email:       email,
		DisplayName: displayName,
		Password:    password,
	}

	ts, err := provide.OpenClient(c.server).Register(ctx, input)
	if err != nil {
		return err
	}

	return ss.
		SetURI(c.server).
		// register token always has an expiry date
		SetExpiresAt(*ts.Token.ExpiresAt).
		SetAccessToken(ts.AccessToken).
		Store()
}

// RegisterRegister helper function to register the register command.
func RegisterRegister(app *kingpin.Application) {
	c := &registerCommand{}

	cmd := app.Command("register", "register a user").
		Action(c.run)

	cmd.Arg("server", "server address").
		Default(provide.DefaultServerURI).
		StringVar(&c.server)
}
