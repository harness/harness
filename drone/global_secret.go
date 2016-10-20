package main

import "github.com/codegangsta/cli"

var globalSecretCmd = cli.Command{
	Name:  "secret",
	Usage: "manage secrets",
	Subcommands: []cli.Command{
		globalSecretAddCmd,
		globalSecretRemoveCmd,
		globalSecretListCmd,
	},
}
