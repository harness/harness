// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"context"
	"encoding/json"
	"os"
	"text/template"
	"time"

	"github.com/harness/gitness/client"

	"github.com/drone/funcmap"
	"gopkg.in/alecthomas/kingpin.v2"
)

const userTmpl = `
uid:   {{ .UID }}
name:  {{ .Name }}
email: {{ .Email }}
admin: {{ .Admin }}
`

type command struct {
	client client.Client
	tmpl   string
	json   bool
}

func (c *command) run(*kingpin.ParseContext) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	user, err := c.client.Self(ctx)
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

// Register the command.
func registerSelf(app *kingpin.CmdClause, client client.Client) {
	c := &command{
		client: client,
	}

	cmd := app.Command("self", "display authenticated user").
		Action(c.run)

	cmd.Flag("json", "json encode the output").
		BoolVar(&c.json)

	cmd.Flag("format", "format the output using a Go template").
		Default(userTmpl).
		Hidden().
		StringVar(&c.tmpl)
}
