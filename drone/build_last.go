package main

import (
	"log"
	"os"
	"text/template"

	"github.com/codegangsta/cli"
)

var buildLastCmd = cli.Command{
	Name:  "last",
	Usage: "show latest build details",
	Action: func(c *cli.Context) {
		if err := buildLast(c); err != nil {
			log.Fatalln(err)
		}
	},
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "format",
			Usage: "format output",
			Value: tmplBuildInfo,
		},
		cli.StringFlag{
			Name:  "branch",
			Usage: "branch name",
			Value: "master",
		},
	},
}

func buildLast(c *cli.Context) error {
	repo := c.Args().First()
	owner, name, err := parseRepo(repo)
	if err != nil {
		return err
	}

	client, err := newClient(c)
	if err != nil {
		return err
	}

	build, err := client.BuildLast(owner, name, c.String("branch"))
	if err != nil {
		return err
	}

	tmpl, err := template.New("_").Parse(c.String("format"))
	if err != nil {
		return err
	}
	return tmpl.Execute(os.Stdout, build)
}
