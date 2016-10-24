package main

import "github.com/codegangsta/cli"

var globalCmd = cli.Command{
	Name:  "global",
	Usage: "manage global state",
	Subcommands: []cli.Command{
		globalSecretCmd,
	},
}
