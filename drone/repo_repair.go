package main

import (
	"github.com/urfave/cli"
)

var repoRepairCmd = cli.Command{
	Name:   "repair",
	Usage:  "repair repository webhooks",
	Action: repoRepair,
}

func repoRepair(c *cli.Context) error {
	repo := c.Args().First()
	owner, name, err := parseRepo(repo)
	if err != nil {
		return err
	}
	client, err := newClient(c)
	if err != nil {
		return err
	}
	return client.RepoRepair(owner, name)
}
