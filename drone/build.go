package main

import "github.com/urfave/cli"

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
		buildApproveCmd,
		buildDeclineCmd,
		buildQueueCmd,
	},
}
