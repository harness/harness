package main

import "github.com/urfave/cli"

var globalSecretCmd = cli.Command{
	Name:  "secret",
	Usage: "manage secrets",
	Subcommands: []cli.Command{
		globalSecretAddCmd,
		globalSecretRemoveCmd,
		globalSecretListCmd,
	},
}
