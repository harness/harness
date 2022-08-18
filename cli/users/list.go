// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package users

import (
	"encoding/json"
	"os"
	"text/template"

	"github.com/drone/funcmap"
	"github.com/harness/scm/cli/util"
	"github.com/harness/scm/types"

	"gopkg.in/alecthomas/kingpin.v2"
)

const userTmpl = `
id:    {{ .ID }}
email: {{ .Email }}
admin: {{ .Admin }}
`

type listCommand struct {
	tmpl string
	page int
	size int
	json bool
}

func (c *listCommand) run(*kingpin.ParseContext) error {
	client, err := util.Client()
	if err != nil {
		return err
	}
	list, err := client.UserList(types.Params{
		Size: c.size,
		Page: c.page,
	})
	if err != nil {
		return err
	}
	tmpl, err := template.New("_").Funcs(funcmap.Funcs).Parse(c.tmpl + "\n")
	if err != nil {
		return err
	}
	if c.json {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(list)
	}
	for _, item := range list {
		tmpl.Execute(os.Stdout, item)
	}
	return nil
}

// helper function registers the user list command
func registerList(app *kingpin.CmdClause) {
	c := new(listCommand)

	cmd := app.Command("ls", "display a list of users").
		Action(c.run)

	cmd.Flag("page", "page number").
		IntVar(&c.page)

	cmd.Flag("per-page", "page size").
		IntVar(&c.size)

	cmd.Flag("json", "json encode the output").
		BoolVar(&c.json)

	cmd.Flag("format", "format the output using a Go template").
		Default(userTmpl).
		Hidden().
		StringVar(&c.tmpl)
}
