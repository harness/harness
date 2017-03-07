package main

import "github.com/codegangsta/cli"

var buildCmd = cli.Command{
	Name:  "build",
	Usage: "manage builds",
	Subcommands: []cli.Command{
		buildListCmd,
		buildLastCmd,
		buildLogsCmd,
		buildInfoCmd,
		buildStopCmd,
		buildStartCmd,
		buildQueueCmd,
	},
}
