// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package execution

import (
	"encoding/json"
	"os"
	"text/template"

	"github.com/drone/funcmap"
	"github.com/harness/scm/cli/util"
	"github.com/harness/scm/types"

	"gopkg.in/alecthomas/kingpin.v2"
)

const executionTmpl = `
id:   {{ .ID }}
slug: {{ .Slug }}
name: {{ .Name }}
desc: {{ .Desc }}
`

type listCommand struct {
	slug string
	tmpl string
	json bool
	page int
	size int
}

func (c *listCommand) run(*kingpin.ParseContext) error {
	client, err := util.Client()
	if err != nil {
		return err
	}
	list, err := client.ExecutionList(c.slug, types.Params{
		Size: c.size,
		Page: c.page,
	})
	if err != nil {
		return err
	}
	if c.json {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(list)
	}
	tmpl, err := template.New("_").Funcs(funcmap.Funcs).Parse(c.tmpl + "\n")
	if err != nil {
		return err
	}
	for _, item := range list {
		tmpl.Execute(os.Stdout, item)
	}
	return nil
}

// helper function registers the list command
func registerList(app *kingpin.CmdClause) {
	c := new(listCommand)

	cmd := app.Command("ls", "display a list of executions").
		Action(c.run)

	cmd.Arg("pipeline ", "pipeline slug").
		Required().
		StringVar(&c.slug)

	cmd.Flag("page", "page number").
		IntVar(&c.page)

	cmd.Flag("per-page", "page size").
		IntVar(&c.size)

	cmd.Flag("json", "json encode the output").
		BoolVar(&c.json)

	cmd.Flag("format", "format the output using a Go template").
		Default(executionTmpl).
		Hidden().
		StringVar(&c.tmpl)
}
