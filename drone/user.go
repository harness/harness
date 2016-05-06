package main

import "github.com/codegangsta/cli"

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
