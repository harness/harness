// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package migrate

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/internal/store/database/migrate"

	"gopkg.in/alecthomas/kingpin.v2"
)

type commandCurrent struct {
	envfile string
}

func (c *commandCurrent) run(*kingpin.ParseContext) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	db, err := getDB(ctx, c.envfile)
	if err != nil {
		return err
	}

	version, err := migrate.Current(ctx, db)
	if err != nil {
		return err
	}

	fmt.Println(version)

	return nil
}

func registerCurrent(app *kingpin.CmdClause) {
	c := &commandCurrent{}

	cmd := app.Command("current", "display the current version of the database").
		Action(c.run)

	cmd.Arg("envfile", "load the environment variable file").
		Default("").
		StringVar(&c.envfile)
}
