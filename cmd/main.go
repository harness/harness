package main

import (
	"github.com/codegangsta/cli"
	"os"
)

var (
	// commit sha for the current build.
	version  string = "0.3-dev"
	revision string
)

func main() {
	app := cli.NewApp()
	app.Name = "drone"
	app.Version = version
	app.Usage = "command line utility"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "t, token",
			Value:  "",
			Usage:  "server auth token",
			EnvVar: "DRONE_TOKEN",
		},
		cli.StringFlag{
			Name:   "s, server",
			Value:  "",
			Usage:  "server location",
			EnvVar: "DRONE_SERVER",
		},
	}

	app.Commands = []cli.Command{
		NewBuildCommand(),
		NewEnableCommand(),
		NewDisableCommand(),
		NewRestartCommand(),
		NewWhoamiCommand(),
	}

	app.Run(os.Args)
}
