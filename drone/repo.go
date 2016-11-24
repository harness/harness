package main

import "github.com/codegangsta/cli"

var repoCmd = cli.Command{
	Name:  "repo",
	Usage: "manage repositories",
	Subcommands: []cli.Command{
		repoListCmd,
		repoInfoCmd,
		repoAddCmd,
		repoRemoveCmd,
		repoChownCmd,
	},
}
