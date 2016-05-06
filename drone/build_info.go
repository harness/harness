package main

import (
	"log"
	"os"
	"strconv"
	"text/template"

	"github.com/codegangsta/cli"
)

var buildInfoCmd = cli.Command{
	Name:  "info",
	Usage: "show build details",
	Action: func(c *cli.Context) {
		if err := buildInfo(c); err != nil {
			log.Fatalln(err)
		}
	},
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "format",
			Usage: "format output",
			Value: tmplBuildInfo,
		},
	},
}

func buildInfo(c *cli.Context) error {
	repo := c.Args().First()
	owner, name, err := parseRepo(repo)
	if err != nil {
		return err
	}
	number, err := strconv.Atoi(c.Args().Get(1))
	if err != nil {
		return err
	}

	client, err := newClient(c)
	if err != nil {
		return err
	}

	build, err := client.Build(owner, name, number)
	if err != nil {
		return err
	}

	tmpl, err := template.New("_").Parse(c.String("format"))
	if err != nil {
		return err
	}
	return tmpl.Execute(os.Stdout, build)
}

// template for build information
var tmplBuildInfo = `Number: {{ .Number }}
Status: {{ .Status }}
Event: {{ .Event }}
Commit: {{ .Commit }}
Branch: {{ .Branch }}
Ref: {{ .Ref }}
Message: {{ .Message }}
Author: {{ .Author }}
`
