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

package user

import (
	"context"
	"encoding/json"
	"os"
	"text/template"
	"time"

	"github.com/harness/gitness/cli/provide"

	"github.com/drone/funcmap"
	"gopkg.in/alecthomas/kingpin.v2"
)

const userTmpl = `
uid:   {{ .UID }}
name:  {{ .DisplayName }}
email: {{ .Email }}
admin: {{ .Admin }}
`

type command struct {
	tmpl string
	json bool
}

func (c *command) run(*kingpin.ParseContext) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	user, err := provide.Client().Self(ctx)
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
func registerSelf(app *kingpin.CmdClause) {
	c := &command{}

	cmd := app.Command("self", "display authenticated user").
		Action(c.run)

	cmd.Flag("json", "json encode the output").
		BoolVar(&c.json)

	cmd.Flag("format", "format the output using a Go template").
		Default(userTmpl).
		Hidden().
		StringVar(&c.tmpl)
}
