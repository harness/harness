// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package users

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/template"
	"time"

	"github.com/harness/gitness/client"
	"github.com/harness/gitness/types"

	"github.com/dchest/uniuri"
	"github.com/drone/funcmap"
	"github.com/gotidy/ptr"
	"gopkg.in/alecthomas/kingpin.v2"
)

type updateCommand struct {
	client  client.Client
	id      string
	email   string
	admin   bool
	demote  bool
	passgen bool
	pass    string
	tmpl    string
	json    bool
}

func (c *updateCommand) run(*kingpin.ParseContext) error {
	in := new(types.UserInput)
	if v := c.email; v != "" {
		in.Email = ptr.String(v)
	}
	if v := c.pass; v != "" {
		in.Password = ptr.String(v)
	}
	if v := c.admin; v {
		in.Admin = ptr.Bool(v)
	}
	if v := c.demote; v {
		in.Admin = ptr.Bool(false)
	}
	if c.passgen {
		const maxRandomChars = 8
		v := uniuri.NewLen(maxRandomChars)
		in.Password = ptr.String(v)
		fmt.Printf("generated temporary password: %s\n", v)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	user, err := c.client.UserUpdate(ctx, c.id, in)
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

// helper function registers the user update command.
func registerUpdate(app *kingpin.CmdClause, client client.Client) {
	c := &updateCommand{
		client: client,
	}

	cmd := app.Command("update", "update a user").
		Action(c.run)

	cmd.Arg("id or email", "user id or email").
		Required().
		StringVar(&c.id)

	cmd.Flag("email", "update user email").
		StringVar(&c.email)

	cmd.Flag("password", "update user password").
		StringVar(&c.pass)

	cmd.Flag("password-gen", "generate and update user password").
		BoolVar(&c.passgen)

	cmd.Flag("promote", "promote user to admin").
		BoolVar(&c.admin)

	cmd.Flag("demote", "demote user from admin").
		BoolVar(&c.demote)

	cmd.Flag("json", "json encode the output").
		BoolVar(&c.json)

	cmd.Flag("format", "format the output using a Go template").
		Default(userTmpl).
		Hidden().
		StringVar(&c.tmpl)
}
