// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	"github.com/harness/gitness/internal/api/controller/user"

	"github.com/drone/funcmap"
	"github.com/gotidy/ptr"
	"gopkg.in/alecthomas/kingpin.v2"
)

const tokenTmpl = `
principalID: {{ .Token.PrincipalID }}
uid:         {{ .Token.UID }}
expiresAt:   {{ .Token.ExpiresAt }}
token:       {{ .AccessToken }}
` //#nosec G101

type createPATCommand struct {
	uid         string
	lifetimeInS int64

	json bool
	tmpl string
}

func (c *createPATCommand) run(*kingpin.ParseContext) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var lifeTime *time.Duration
	if c.lifetimeInS > 0 {
		lifeTime = ptr.Duration(time.Duration(int64(time.Second) * c.lifetimeInS))
	}

	in := user.CreateTokenInput{
		UID:      c.uid,
		Lifetime: lifeTime,
	}

	tokenResp, err := provide.Client().UserCreatePAT(ctx, in)
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
	c := &createPATCommand{}

	cmd := app.Command("pat", "create personal access token").
		Action(c.run)

	cmd.Arg("uid", "the uid of the token").
		Required().StringVar(&c.uid)

	cmd.Arg("lifetime", "the lifetime of the token in seconds").
		Int64Var(&c.lifetimeInS)

	cmd.Flag("json", "json encode the output").
		BoolVar(&c.json)

	cmd.Flag("format", "format the output using a Go template").
		Default(tokenTmpl).
		Hidden().
		StringVar(&c.tmpl)
}
