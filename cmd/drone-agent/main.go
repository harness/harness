package main

import (
	"fmt"
	"os"

	"github.com/drone/drone/version"

	_ "github.com/joho/godotenv/autoload"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "drone-agent"
	app.Version = version.Version.String()
	app.Usage = "drone agent"
	app.Action = loop
	app.Commands = []cli.Command{
		{
			Name:   "ping",
			Usage:  "ping the agent",
			Action: pinger,
		},
	}
	app.Flags = []cli.Flag{
		cli.BoolTFlag{
			EnvVar: "DRONE_DEBUG",
			Name:   "debug",
			Usage:  "start the agent in debug mode",
		},
		cli.BoolTFlag{
			EnvVar: "DRONE_DEBUG_NOCOLOR",
			Name:   "nocolor",
			Usage:  "disable colored debug output",
		},
		cli.BoolFlag{
			EnvVar: "DRONE_DEBUG_PRETTY",
			Name:   "pretty",
			Usage:  "enable pretty-printed debug output",
		},
		cli.StringFlag{
			EnvVar: "DRONE_FILTER",
			Name:   "filter",
			Usage:  "filter expression used to restrict builds by label",
		},
		cli.BoolTFlag{
			EnvVar: "DRONE_HEALTHCHECK",
			Name:   "healthcheck",
			Usage:  "enables the healthcheck endpoint",
		},
		cli.StringFlag{
			EnvVar: "DRONE_HOSTNAME,HOSTNAME",
			Name:   "hostname",
		},
		cli.IntFlag{
			EnvVar: "DRONE_MAX_PROCS",
			Name:   "max-procs",
			Value:  1,
		},
		cli.StringFlag{
			EnvVar: "DRONE_PASSWORD,DRONE_SECRET",
			Name:   "password",
			Usage:  "drone auth password",
		},
		cli.StringFlag{
			EnvVar: "DRONE_PLATFORM",
			Name:   "platform",
			Value:  "linux/amd64",
		},
		cli.StringFlag{
			EnvVar: "DRONE_SERVER",
			Name:   "server",
			Usage:  "drone server address",
			Value:  "localhost:9000",
		},
		cli.StringFlag{
			EnvVar: "DRONE_USERNAME",
			Name:   "username",
			Usage:  "drone auth username",
			Value:  "x-oauth-basic",
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
