package main

import (
	"log"
	"os"
	"text/template"

	"github.com/codegangsta/cli"
)

var buildListCmd = cli.Command{
	Name:  "list",
	Usage: "show build history",
	Action: func(c *cli.Context) {
		if err := buildList(c); err != nil {
			log.Fatalln(err)
		}
	},
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "format",
			Usage: "format output",
			Value: tmplBuildList,
		},
		cli.StringFlag{
			Name:  "branch",
			Usage: "branch filter",
		},
		cli.StringFlag{
			Name:  "event",
			Usage: "event filter",
		},
		cli.StringFlag{
			Name:  "status",
			Usage: "status filter",
		},
		cli.IntFlag{
			Name:  "limit",
			Usage: "limit the list size",
			Value: 25,
		},
	},
}

func buildList(c *cli.Context) error {
	repo := c.Args().First()
	owner, name, err := parseRepo(repo)
	if err != nil {
		return err
	}

	client, err := newClient(c)
	if err != nil {
		return err
	}

	builds, err := client.BuildList(owner, name)
	if err != nil {
		return err
	}

	tmpl, err := template.New("_").Parse(c.String("format") + "\n")
	if err != nil {
		return err
	}

	branch := c.String("branch")
	event := c.String("event")
	status := c.String("status")
	limit := c.Int("limit")

	var count int
	for _, build := range builds {
		if count >= limit {
			break
		}
		if branch != "" && build.Branch != branch {
			continue
		}
		if event != "" && build.Event != event {
			continue
		}
		if status != "" && build.Status != status {
			continue
		}
		tmpl.Execute(os.Stdout, build)
		count++
	}
	return nil
}

// template for build list information
var tmplBuildList = "\x1b[33mBuild #{{ .Number }} \x1b[0m" + `
Status: {{ .Status }}
Event: {{ .Event }}
Commit: {{ .Commit }}
Branch: {{ .Branch }}
Ref: {{ .Ref }}
Author: {{ .Author }} {{ if .Email }}<{{.Email}}>{{ end }}
Message: {{ .Message }}
`
