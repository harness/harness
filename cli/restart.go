package main

import (
	"github.com/codegangsta/cli"
	"github.com/drone/drone/client"
)

// NewRestartCommand returns the CLI command for "restart".
func NewRestartCommand() cli.Command {
	return cli.Command{
		Name:  "restart",
		Usage: "restarts a build on the server",
		Flags: []cli.Flag{},
		Action: func(c *cli.Context) {
			handle(c, restartCommandFunc)
		},
	}
}

// restartCommandFunc executes the "restart" command.
func restartCommandFunc(c *cli.Context, client *client.Client) error {
	var host, owner, repo, branch, sha string
	var args = c.Args()

	if len(args) != 0 {
		host, owner, repo = parseRepo(args[0])
	}

	switch len(args) {
	case 2:
		branch = "master"
		sha = args[1]
	case 3, 4, 5:
		branch = args[1]
		sha = args[2]
	}

	return client.Commits.Rebuild(host, owner, repo, branch, sha)
}
