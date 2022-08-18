// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pipeline

import (
	"encoding/json"
	"os"
	"text/template"

	"github.com/gotidy/ptr"
	"github.com/harness/scm/cli/util"
	"github.com/harness/scm/types"

	"github.com/drone/funcmap"
	"gopkg.in/alecthomas/kingpin.v2"
)

type updateCommand struct {
	slug string
	name string
	desc string
	tmpl string
	json bool
}

func (c *updateCommand) run(*kingpin.ParseContext) error {
	client, err := util.Client()
	if err != nil {
		return err
	}

	in := new(types.PipelineInput)
	if v := c.name; v != "" {
		in.Name = ptr.String(v)
	}
	if v := c.desc; v != "" {
		in.Desc = ptr.String(v)
	}

	item, err := client.PipelineUpdate(c.slug, in)
	if err != nil {
		return err
	}
	if c.json {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(item)
	}
	tmpl, err := template.New("_").Funcs(funcmap.Funcs).Parse(c.tmpl)
	if err != nil {
		return err
	}
	return tmpl.Execute(os.Stdout, item)
}

// helper function registers the user update command
func registerUpdate(app *kingpin.CmdClause) {
	c := new(updateCommand)

	cmd := app.Command("update", "update a pipeline").
		Action(c.run)

	cmd.Arg("slug", "pipeline slug").
		Required().
		StringVar(&c.slug)

	cmd.Flag("name", "update pipeline name").
		StringVar(&c.name)

	cmd.Flag("desc", "update pipeline description").
		StringVar(&c.desc)

	cmd.Flag("json", "json encode the output").
		BoolVar(&c.json)

	cmd.Flag("format", "format the output using a Go template").
		Default(pipelineTmpl).
		Hidden().
		StringVar(&c.tmpl)
}
