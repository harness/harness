package main

import (
	"os"
	"text/template"

	"github.com/urfave/cli"
)

var infoCmd = cli.Command{
	Name:   "info",
	Usage:  "show information about the current user",
	Action: info,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "format",
			Usage: "format output",
			Value: tmplUserInfo,
		},
	},
}

func info(c *cli.Context) error {
	client, err := newClient(c)
	if err != nil {
		return err
	}

	user, err := client.Self()
	if err != nil {
		return err
	}

	tmpl, err := template.New("_").Parse(c.String("format") + "\n")
	if err != nil {
		return err
	}

	return tmpl.Execute(os.Stdout, user)
}

// template for user information
var tmplInfo = `User: {{ .Login }}
Email: {{ .Email }}`
