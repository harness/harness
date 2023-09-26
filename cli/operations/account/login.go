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
	"github.com/harness/gitness/cli/textui"

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
