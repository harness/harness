package main

import (
	"github.com/codegangsta/cli"
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
func disableCommandFunc(c *cli.Context, client *Client) error {
	var repo string
	var arg = c.Args()

	if len(arg) != 0 {
		repo = arg[0]
	}

	err := client.Do("DELETE", "/v1/repos/"+repo, nil, nil)
	if err != nil {
		return err
	}

	return nil
}
