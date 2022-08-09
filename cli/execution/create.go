// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package execution

import (
	"encoding/json"
	"os"
	"text/template"

	"github.com/bradrydzewski/my-app/cli/util"
	"github.com/bradrydzewski/my-app/types"

	"github.com/drone/funcmap"
	"gopkg.in/alecthomas/kingpin.v2"
)

type createCommand struct {
	pipeline string
	slug     string
	name     string
	desc     string
	tmpl     string
	json     bool
}

func (c *createCommand) run(*kingpin.ParseContext) error {
	client, err := util.Client()
	if err != nil {
		return err
	}
	in := &types.Execution{
		Slug: c.slug,
		Name: c.name,
		Desc: c.desc,
	}
	item, err := client.ExecutionCreate(c.pipeline, in)
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

// helper function registers the user create command
func registerCreate(app *kingpin.CmdClause) {
	c := new(createCommand)

	cmd := app.Command("create", "create a execution").
		Action(c.run)

	cmd.Arg("pipeline ", "pipeline slug").
		Required().
		StringVar(&c.pipeline)

	cmd.Arg("slug ", "execution slug").
		Required().
		StringVar(&c.slug)

	cmd.Flag("name", "execution name").
		StringVar(&c.name)

	cmd.Flag("desc", "execution description").
		StringVar(&c.desc)

	cmd.Flag("json", "json encode the output").
		BoolVar(&c.json)

	cmd.Flag("format", "format the output using a Go template").
		Default(executionTmpl).
		Hidden().
		StringVar(&c.tmpl)
}
