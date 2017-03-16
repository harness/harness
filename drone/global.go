package main

import "github.com/urfave/cli"

var globalCmd = cli.Command{
	Name:  "global",
	Usage: "manage global state",
	Subcommands: []cli.Command{
		globalSecretCmd,
	},
}
