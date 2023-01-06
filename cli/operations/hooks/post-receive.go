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

type postReceiveCommand struct{}

func (c *postReceiveCommand) run(*kingpin.ParseContext) error {
	cli, err := githook.NewCLI()
	if err != nil {
		return fmt.Errorf("failed to create githook cli: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	return cli.PostReceive(ctx)
}

func registerPostReceive(app *kingpin.CmdClause) {
	c := &postReceiveCommand{}

	app.Command("post-receive", "hook that is executed after all references of the push got updated").
		Action(c.run)
}
