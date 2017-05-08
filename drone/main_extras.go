// +build extras

package main

import (
	"fmt"
	"os"

	"github.com/drone/drone/drone/agent"
	"github.com/drone/drone/drone/build"
	"github.com/drone/drone/drone/deploy"
	"github.com/drone/drone/drone/exec"
	"github.com/drone/drone/drone/info"
	"github.com/drone/drone/drone/registry"
	"github.com/drone/drone/drone/repo"
	"github.com/drone/drone/drone/secret"
	"github.com/drone/drone/drone/user"
	"github.com/drone/drone/version"

	"github.com/drone/drone/extras/cmd/drone/server"

	"github.com/ianschenck/envflag"
	_ "github.com/joho/godotenv/autoload"
	"github.com/urfave/cli"
)

func main() {
	envflag.Parse()

	app := cli.NewApp()
	app.Name = "drone"
	app.Version = version.Version.String()
	app.Usage = "command line utility"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "t, token",
			Usage:  "server auth token",
			EnvVar: "DRONE_TOKEN",
		},
		cli.StringFlag{
			Name:   "s, server",
			Usage:  "server location",
			EnvVar: "DRONE_SERVER",
		},
	}
	app.Commands = []cli.Command{
		agent.Command,
		build.Command,
		deploy.Command,
		exec.Command,
		info.Command,
		registry.Command,
		secret.Command,
		server.Command,
		repo.Command,
		user.Command,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
