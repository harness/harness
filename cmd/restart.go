package main

import (
	"fmt"

	"github.com/codegangsta/cli"
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
func restartCommandFunc(c *cli.Context, client *Client) error {
	var branch string = "master"
	var commit string
	var repo string
	var arg = c.Args()

	switch len(arg) {
	case 2:
		repo = arg[0]
		commit = arg[1]
	case 3:
		repo = arg[0]
		branch = arg[1]
		commit = arg[2]
	}

	path := fmt.Sprintf("/v1/repos/%s/branches/%s/commits/%s?action=rebuild", repo, branch, commit)
	return client.Do("POST", path, nil, nil)
}
