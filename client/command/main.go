package main

import (
	"github.com/codegangsta/cli"
	"os"
)

func main() {
	app := cli.NewApp()
	app.Name = "drone"
	app.Version = "1.0"
	app.Usage = "command line utility"
	app.EnableBashCompletion = true

	app.Commands = []cli.Command{
		NewEnableCommand(),
		NewDisableCommand(),
		NewRestartCommand(),
		NewWhoamiCommand(),
	}

	app.Run(os.Args)
}
