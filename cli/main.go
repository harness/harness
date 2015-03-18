package main

import (
	"os"

	"errors"
	"fmt"
	"github.com/codegangsta/cli"
	"net/url"
)

var (
	// commit sha for the current build.
	version  string
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
			Usage:  "server location (e.g. --server=http://test.drone.io)",
			EnvVar: "DRONE_SERVER",
		},
	}

	app.Commands = []cli.Command{
		NewBuildCommand(),
		NewReposCommand(),
		NewStatusCommand(),
		NewEnableCommand(),
		NewDisableCommand(),
		NewRestartCommand(),
		NewWhoamiCommand(),
		NewSetKeyCommand(),
		NewDeleteCommand(),
	}

	app.Before = func(c *cli.Context) error {
		f := c.GlobalString("server")

		if f == "" {
			return nil
		}

		uri, err := url.Parse(f)

		if err != nil {
			return err
		}

		if uri.Scheme == "" {
			fmt.Println("-s/--server requires a scheme (e.g. http://)")
			return errors.New("Invalid host provided for server")
		}

		return nil
	}

	app.Run(os.Args)
}
