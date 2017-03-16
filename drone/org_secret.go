package main

import "github.com/urfave/cli"

var orgSecretCmd = cli.Command{
	Name:  "secret",
	Usage: "manage secrets",
	Subcommands: []cli.Command{
		orgSecretAddCmd,
		orgSecretRemoveCmd,
		orgSecretListCmd,
	},
}
