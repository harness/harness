package main

import (
	"github.com/codegangsta/cli"
	"github.com/drone/drone/client"
)

// NewDisableCommand returns the CLI command for "disable".
func NewDisableCommand() cli.Command {
	return cli.Command{
		Name:  "disable",
		Usage: "disable a repository",
		Flags: []cli.Flag{},
		Action: func(c *cli.Context) {
			handle(c, disableCommandFunc)
		},
	}
}

// disableCommandFunc executes the "disable" command.
func disableCommandFunc(c *cli.Context, client *client.Client) error {
	var host, owner, name string
	var args = c.Args()

	if len(args) != 0 {
		host, owner, name = parseRepo(args[0])
	}

	return client.Repos.Disable(host, owner, name)
}
