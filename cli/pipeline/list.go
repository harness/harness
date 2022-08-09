// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pipeline

import (
	"encoding/json"
	"os"
	"text/template"

	"github.com/bradrydzewski/my-app/cli/util"
	"github.com/bradrydzewski/my-app/types"
	"github.com/drone/funcmap"

	"gopkg.in/alecthomas/kingpin.v2"
)

const pipelineTmpl = `
id:   {{ .ID }}
slug: {{ .Slug }}
name: {{ .Name }}
desc: {{ .Desc }}
`

type listCommand struct {
	page int
	size int
	json bool
	tmpl string
}

func (c *listCommand) run(*kingpin.ParseContext) error {
	client, err := util.Client()
	if err != nil {
		return err
	}
	list, err := client.PipelineList(types.Params{
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

// helper function registers the user list command
func registerList(app *kingpin.CmdClause) {
	c := new(listCommand)

	cmd := app.Command("ls", "display a list of pipelines").
		Action(c.run)

	cmd.Flag("page", "page number").
		IntVar(&c.page)

	cmd.Flag("per-page", "page size").
		IntVar(&c.size)

	cmd.Flag("json", "json encode the output").
		BoolVar(&c.json)

	cmd.Flag("format", "format the output using a Go template").
		Default(pipelineTmpl).
		Hidden().
		StringVar(&c.tmpl)
}
