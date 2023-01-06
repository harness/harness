// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package hooks

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/internal/githook"

	"gopkg.in/alecthomas/kingpin.v2"
)

type updateCommand struct {
	ref string
	old string
	new string
}

func (c *updateCommand) run(*kingpin.ParseContext) error {
	cli, err := githook.NewCLI()
	if err != nil {
		return fmt.Errorf("failed to create githook cli: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	return cli.Update(ctx, c.ref, c.old, c.new)
}

func registerUpdate(app *kingpin.CmdClause) {
	c := &updateCommand{}

	cmd := app.Command("update", "hook that is executed before the specific reference gets updated").
		Action(c.run)

	cmd.Arg("ref", "reference for which the hook is executed").
		Required().
		StringVar(&c.ref)

	cmd.Arg("old", "old commit sha").
		Required().
		StringVar(&c.old)

	cmd.Arg("new", "new commit sha").
		Required().
		StringVar(&c.new)
}
