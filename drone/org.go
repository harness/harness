package main

import "github.com/codegangsta/cli"

var orgCmd = cli.Command{
	Name:  "org",
	Usage: "manage organizations",
	Subcommands: []cli.Command{
		orgSecretCmd,
	},
}
