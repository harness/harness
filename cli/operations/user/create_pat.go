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

	"github.com/harness/gitness/cli/util"
	"github.com/harness/gitness/internal/api/controller/user"
	"github.com/harness/gitness/types/enum"

	"github.com/drone/funcmap"
	"gopkg.in/alecthomas/kingpin.v2"
)

const tokenTmpl = `
name:      {{ .Token.Name }}
expiresAt: {{ .Token.ExpiresAt }}
token:     {{ .AccessToken }}
` //#nosec G101

type createPATCommand struct {
	name        string
	lifetimeInS int64

	json bool
	tmpl string
}

func (c *createPATCommand) run(*kingpin.ParseContext) error {
	client, err := util.Client()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	in := user.CreateTokenInput{
		Name:     c.name,
		Lifetime: time.Duration(int64(time.Second) * c.lifetimeInS),
		Grants:   enum.AccessGrantAll,
	}

	tokenResp, err := client.UserCreatePAT(ctx, in)
	if err != nil {
		return err
	}
	if c.json {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(tokenResp)
	}
	tmpl, err := template.New("_").Funcs(funcmap.Funcs).Parse(c.tmpl)
	if err != nil {
		return err
	}
	return tmpl.Execute(os.Stdout, tokenResp)
}

// Register the command.
func registerCreatePAT(app *kingpin.CmdClause) {
	c := new(createPATCommand)

	cmd := app.Command("pat", "create personal access token").
		Action(c.run)

	cmd.Arg("name", "the name of the token").
		Required().StringVar(&c.name)

	cmd.Arg("lifetime", "the lifetime of the token in seconds").
		Required().Int64Var(&c.lifetimeInS)

	cmd.Flag("json", "json encode the output").
		BoolVar(&c.json)

	cmd.Flag("format", "format the output using a Go template").
		Default(tokenTmpl).
		Hidden().
		StringVar(&c.tmpl)
}
