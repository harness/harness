package user

import "github.com/urfave/cli"

// Command exports the user command set.
var Command = cli.Command{
	Name:  "user",
	Usage: "manage users",
	Subcommands: []cli.Command{
		userListCmd,
		userInfoCmd,
		userAddCmd,
		userRemoveCmd,
	},
}
