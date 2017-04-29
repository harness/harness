package repo

import (
	"github.com/drone/drone/drone/internal"
	"github.com/urfave/cli"
)

var repoRepairCmd = cli.Command{
	Name:   "repair",
	Usage:  "repair repository webhooks",
	Action: repoRepair,
}

func repoRepair(c *cli.Context) error {
	repo := c.Args().First()
	owner, name, err := internal.ParseRepo(repo)
	if err != nil {
		return err
	}
	client, err := internal.NewClient(c)
	if err != nil {
		return err
	}
	return client.RepoRepair(owner, name)
}
