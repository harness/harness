// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package hooks

import (
	"github.com/harness/gitness/client"

	"github.com/rs/zerolog/log"
	"gopkg.in/alecthomas/kingpin.v2"
)

type postReceiveCommand struct {
	client client.Client
}

func (c *postReceiveCommand) run(*kingpin.ParseContext) error {
	// TODO: need to implement this method further to completely execute to hooks
	updatedRefsFromGit, err := getUpdatedReferencesFromStdIn()
	if err != nil {
		return err
	}
	for _, ref := range updatedRefsFromGit {
		log.Info().Msgf("This is the post-receive hook, ref: %s, old commit sha: %s, new commit sha: %s",
			ref.branch, ref.oldCommit, ref.newCommit)
	}
	return nil
}

func registerPostReceive(app *kingpin.CmdClause, client client.Client) {
	c := &postReceiveCommand{
		client: client,
	}

	app.Command("post-receive", "hook that is executed just after all the refs are updated").
		Action(c.run)
}
