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

package migrate

import (
	"context"
	"time"

	"github.com/harness/gitness/app/store/database/migrate"

	"gopkg.in/alecthomas/kingpin.v2"
)

type commandTo struct {
	envfile string
	version string
}

func (c *commandTo) run(_ *kingpin.ParseContext) error {
	ctx := setupLoggingContext(context.Background())
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	db, err := getDB(ctx, c.envfile)
	if err != nil {
		return err
	}

	return migrate.To(ctx, db, c.version)
}

func registerTo(app *kingpin.CmdClause) {
	c := &commandTo{}

	cmd := app.Command("to", "migrates the database to the provided version").
		Action(c.run)

	cmd.Arg("version", "database version to migrate to").
		Required().
		StringVar(&c.version)

	cmd.Arg("envfile", "load the environment variable file").
		Default("").
		StringVar(&c.envfile)
}
