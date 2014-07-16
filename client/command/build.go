package main

import (
	"github.com/codegangsta/cli"
)

// NewBuildCommand returns the CLI command for "build".
func NewBuildCommand() cli.Command {
	return cli.Command{
		Name:  "build",
		Usage: "run a local build",
		Flags: []cli.Flag{},
		Action: func(c *cli.Context) {
			buildCommandFunc(c)
		},
	}
}

// buildCommandFunc executes the "build" command.
func buildCommandFunc(c *cli.Context) error {

	return nil
}
