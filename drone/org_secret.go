package main

import "github.com/codegangsta/cli"

var orgSecretCmd = cli.Command{
	Name:  "secret",
	Usage: "manage secrets",
	Subcommands: []cli.Command{
		orgSecretAddCmd,
		orgSecretRemoveCmd,
		orgSecretListCmd,
	},
}
