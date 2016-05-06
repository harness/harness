package main

import (
	"log"
	"os"
	"text/template"

	"github.com/codegangsta/cli"
)

var repoInfoCmd = cli.Command{
	Name:  "info",
	Usage: "show repository details",
	Action: func(c *cli.Context) {
		if err := repoInfo(c); err != nil {
			log.Fatalln(err)
		}
	},
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "format",
			Usage: "format output",
			Value: tmplRepoInfo,
		},
	},
}

func repoInfo(c *cli.Context) error {
	arg := c.Args().First()
	owner, name, err := parseRepo(arg)
	if err != nil {
		return err
	}

	client, err := newClient(c)
	if err != nil {
		return err
	}

	repo, err := client.Repo(owner, name)
	if err != nil {
		return err
	}

	tmpl, err := template.New("_").Parse(c.String("format"))
	if err != nil {
		return err
	}
	return tmpl.Execute(os.Stdout, repo)
}

// template for repo information
var tmplRepoInfo = `Owner: {{ .Owner }}
Repo: {{ .Name }}
Type: {{ .Kind }}
Private: {{ .IsPrivate }}
Remote: {{ .Clone }}
`
