package main

import (
	"github.com/codegangsta/cli"
	"github.com/drone/drone/client"
)

// NewEnableCommand returns the CLI command for "enable".
func NewEnableCommand() cli.Command {
	return cli.Command{
		Name:  "enable",
		Usage: "enable a repository",
		Flags: []cli.Flag{},
		Action: func(c *cli.Context) {
			handle(c, enableCommandFunc)
		},
	}
}

// enableCommandFunc executes the "enable" command.
func enableCommandFunc(c *cli.Context, client *client.Client) error {
	var host, owner, name string
	var args = c.Args()

	if len(args) != 0 {
		host, owner, name = parseRepo(args[0])
	}

	return client.Repos.Enable(host, owner, name)
}
