package repo

import "github.com/urfave/cli"

// Command exports the repository command.
var Command = cli.Command{
	Name:  "repo",
	Usage: "manage repositories",
	Subcommands: []cli.Command{
		repoListCmd,
		repoInfoCmd,
		repoAddCmd,
		repoUpdateCmd,
		repoRemoveCmd,
		repoRepairCmd,
		repoChownCmd,
	},
}
