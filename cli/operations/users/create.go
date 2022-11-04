// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package users

import (
	"context"
	"encoding/json"
	"os"
	"text/template"
	"time"

	"github.com/harness/gitness/client"

	"github.com/harness/gitness/cli/textui"

	"github.com/harness/gitness/types"

	"github.com/drone/funcmap"
	"gopkg.in/alecthomas/kingpin.v2"
)

type createCommand struct {
	client client.Client
	email  string
	admin  bool
	tmpl   string
	json   bool
}

func (c *createCommand) run(*kingpin.ParseContext) error {
	in := &types.User{
		Admin:    c.admin,
		Email:    c.email,
		Password: textui.Password(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	user, err := c.client.UserCreate(ctx, in)
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
func registerCreate(app *kingpin.CmdClause, client client.Client) {
	c := &createCommand{
		client: client,
	}

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
