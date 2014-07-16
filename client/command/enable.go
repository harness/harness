package main

import (
	"github.com/codegangsta/cli"
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
func enableCommandFunc(c *cli.Context, client *Client) error {
	var repo string
	var arg = c.Args()

	if len(arg) != 0 {
		repo = arg[0]
	}

	err := client.Do("POST", "/v1/repos/"+repo, nil, nil)
	if err != nil {
		return err
	}

	return nil
}
