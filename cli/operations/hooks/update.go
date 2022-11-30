// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package hooks

import (
	"github.com/harness/gitness/client"

	"github.com/rs/zerolog/log"
	"gopkg.in/alecthomas/kingpin.v2"
)

type updateCommand struct {
	client    client.Client
	branch    string
	oldCommit string
	newCommit string
}

func (c *updateCommand) run(*kingpin.ParseContext) error {
	// TODO: need to implement this method further to completely execute to hooks
	log.Info().Msgf("This is the update hook, ref: %s, old commit sha: %s, new commit sha: %s",
		c.branch, c.oldCommit, c.newCommit)
	return nil
}

func registerUpdate(app *kingpin.CmdClause, client client.Client) {
	c := &updateCommand{
		client: client,
	}

	cmd := app.Command("update", "hook that is executed just before the ref is updated").
		Action(c.run)

	cmd.Arg("ref", "ref on which the hook is executed").
		Required().
		StringVar(&c.branch)

	cmd.Arg("old-commit", "old commit sha").
		Required().
		StringVar(&c.oldCommit)

	cmd.Arg("new-commit", "new commit sha").
		Required().
		StringVar(&c.newCommit)
}
