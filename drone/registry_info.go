package main

import (
	"html/template"
	"os"

	"github.com/urfave/cli"
)

var registryInfoCmd = cli.Command{
	Name:   "info",
	Usage:  "display registry info",
	Action: registryInfo,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "repository",
			Usage: "repository name (e.g. octocat/hello-world)",
		},
		cli.StringFlag{
			Name:  "hostname",
			Usage: "registry hostname",
			Value: "docker.io",
		},
		cli.StringFlag{
			Name:   "format",
			Usage:  "repository name (e.g. octocat/hello-world)",
			Value:  tmplRegistryList,
			Hidden: true,
		},
	},
}

func registryInfo(c *cli.Context) error {
	var (
		hostname = c.String("hostname")
		reponame = c.String("repository")
		format   = c.String("format") + "\n"
	)
	if reponame == "" {
		reponame = c.Args().First()
	}
	owner, name, err := parseRepo(reponame)
	if err != nil {
		return err
	}
	client, err := newClient(c)
	if err != nil {
		return err
	}
	registry, err := client.Registry(owner, name, hostname)
	if err != nil {
		return err
	}
	tmpl, err := template.New("_").Parse(format)
	if err != nil {
		return err
	}
	return tmpl.Execute(os.Stdout, registry)
}
