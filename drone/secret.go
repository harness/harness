package main

import "github.com/codegangsta/cli"

var secretCmd = cli.Command{
	Name:  "secret",
	Usage: "manage secrets",
	Subcommands: []cli.Command{
		secretAddCmd,
		secretRemoveCmd,
		secretListCmd,
	},
}
