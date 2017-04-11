package main

import "github.com/urfave/cli"

var secretCmd = cli.Command{
	Name:  "secret",
	Usage: "manage secrets",
	Subcommands: []cli.Command{
		secretCreateCmd,
		secretDeleteCmd,
		secretUpdateCmd,
		secretInfoCmd,
		secretListCmd,
	},
}
