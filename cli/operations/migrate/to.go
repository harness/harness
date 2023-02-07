// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package migrate

import (
	"context"
	"time"

	"github.com/harness/gitness/internal/store/database/migrate"

	"gopkg.in/alecthomas/kingpin.v2"
)

type commandTo struct {
	envfile string
	version string
}

func (c *commandTo) run(k *kingpin.ParseContext) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
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
