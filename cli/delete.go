package main

import (
	"github.com/codegangsta/cli"
	"github.com/drone/drone/client"
)

// NewDeleteCommand returns the CLI command for "delete".
func NewDeleteCommand() cli.Command {
	return cli.Command{
		Name:  "delete",
		Usage: "delete a repository",
		Flags: []cli.Flag{},
		Action: func(c *cli.Context) {
			handle(c, deleteCommandFunc)
		},
	}
}

// deleteCommandFunc executes the "delete" command.
func deleteCommandFunc(c *cli.Context, client *client.Client) error {
	var host, owner, name string
	var args = c.Args()

	if len(args) != 0 {
		host, owner, name = parseRepo(args[0])
	}

	return client.Repos.Delete(host, owner, name)
}
