package main

import (
	"os"

	"github.com/drone/drone/drone/agent"
	"github.com/drone/drone/drone/server"
	"github.com/drone/drone/version"

	"github.com/codegangsta/cli"
	"github.com/ianschenck/envflag"
	_ "github.com/joho/godotenv/autoload"
)

func main2() {
	envflag.Parse()

	app := cli.NewApp()
	app.Name = "drone"
	app.Version = version.Version
	app.Usage = "command line utility"

	app.Commands = []cli.Command{
		agent.AgentCmd,
		server.ServeCmd,
	}

	app.Run(os.Args)
}
