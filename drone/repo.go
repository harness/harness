package main

import "github.com/urfave/cli"

var repoCmd = cli.Command{
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
