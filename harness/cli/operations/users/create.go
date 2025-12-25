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

package users

import (
	"context"
	"encoding/json"
	"os"
	"text/template"
	"time"

	"github.com/harness/gitness/cli/provide"
	"github.com/harness/gitness/cli/textui"
	"github.com/harness/gitness/types"

	"github.com/drone/funcmap"
	"gopkg.in/alecthomas/kingpin.v2"
)

type createCommand struct {
	email string
	admin bool
	tmpl  string
	json  bool
}

func (c *createCommand) run(*kingpin.ParseContext) error {
	in := &types.User{
		Admin:    c.admin,
		Email:    c.email,
		Password: textui.Password(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	user, err := provide.Client().UserCreate(ctx, in)
	if err != nil {
		return err
	}
	if c.json {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(user)
	}
	tmpl, err := template.New("_").Funcs(funcmap.Funcs).Parse(c.tmpl)
	if err != nil {
		return err
	}
	return tmpl.Execute(os.Stdout, user)
}

// helper function registers the user create command.
func registerCreate(app *kingpin.CmdClause) {
	c := &createCommand{}

	cmd := app.Command("create", "create a user").
		Action(c.run)

	cmd.Arg("email", "user email").
		Required().
		StringVar(&c.email)

	cmd.Arg("admin", "user is admin").
		BoolVar(&c.admin)

	cmd.Flag("json", "json encode the output").
		BoolVar(&c.json)

	cmd.Flag("format", "format the output using a Go template").
		Default(userTmpl).
		Hidden().
		StringVar(&c.tmpl)
}
