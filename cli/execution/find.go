// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package execution

import (
	"encoding/json"
	"os"
	"text/template"

	"github.com/bradrydzewski/my-app/cli/util"

	"github.com/drone/funcmap"
	"gopkg.in/alecthomas/kingpin.v2"
)

type findCommand struct {
	pipeline string
	slug     string
	tmpl     string
	json     bool
}

func (c *findCommand) run(*kingpin.ParseContext) error {
	client, err := util.Client()
	if err != nil {
		return err
	}
	item, err := client.Execution(c.pipeline, c.slug)
	if err != nil {
		return err
	}
	if c.json {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(item)
	}
	tmpl, err := template.New("_").Funcs(funcmap.Funcs).Parse(c.tmpl + "\n")
	if err != nil {
		return err
	}
	return tmpl.Execute(os.Stdout, item)
}

// helper function registers the user find command
func registerFind(app *kingpin.CmdClause) {
	c := new(findCommand)

	cmd := app.Command("find", "display pipeline details").
		Action(c.run)

	cmd.Arg("pipeline ", "pipeline slug").
		Required().
		StringVar(&c.pipeline)

	cmd.Arg("slug ", "execution slug").
		Required().
		StringVar(&c.slug)

	cmd.Flag("json", "json encode the output").
		BoolVar(&c.json)

	cmd.Flag("format", "format the output using a Go template").
		Default(executionTmpl).
		Hidden().
		StringVar(&c.tmpl)
}
