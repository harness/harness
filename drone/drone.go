package main

import (
	"os"

	"github.com/drone/drone/drone/agent"
	"github.com/drone/drone/version"

	"github.com/codegangsta/cli"
	"github.com/ianschenck/envflag"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	envflag.Parse()

	app := cli.NewApp()
	app.Name = "drone"
	app.Version = version.Version
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
		agent.AgentCmd,
		DaemonCmd,
		SignCmd,
		SecretCmd,
	}

	app.Run(os.Args)
}
