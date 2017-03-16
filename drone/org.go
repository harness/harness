package main

import "github.com/urfave/cli"

var orgCmd = cli.Command{
	Name:  "org",
	Usage: "manage organizations",
	Subcommands: []cli.Command{
		orgSecretCmd,
	},
}
