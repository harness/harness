package main

import "github.com/urfave/cli"

var userCmd = cli.Command{
	Name:  "user",
	Usage: "manage users",
	Subcommands: []cli.Command{
		userListCmd,
		userInfoCmd,
		userAddCmd,
		userRemoveCmd,
	},
}
