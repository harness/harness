package main

import (
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/codegangsta/cli"
)

var orgSecretListCmd = cli.Command{
	Name:  "ls",
	Usage: "list all secrets",
	Action: func(c *cli.Context) {
		if err := orgSecretList(c); err != nil {
			log.Fatalln(err)
		}
	},
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "format",
			Usage: "format output",
			Value: tmplOrgSecretList,
		},
		cli.StringFlag{
			Name:  "image",
			Usage: "filter by image",
		},
		cli.StringFlag{
			Name:  "event",
			Usage: "filter by event",
		},
	},
}

func orgSecretList(c *cli.Context) error {
	if len(c.Args()) != 1 {
		cli.ShowSubcommandHelp(c)
		return nil
	}

	team := c.Args().First()

	client, err := newClient(c)
	if err != nil {
		return err
	}

	secrets, err := client.TeamSecretList(team)

	if err != nil || len(secrets) == 0 {
		return err
	}

	tmpl, err := template.New("_").Funcs(orgSecretFuncMap).Parse(c.String("format") + "\n")

	if err != nil {
		return err
	}

	for _, secret := range secrets {
		if c.String("image") != "" && !stringInSlice(c.String("image"), secret.Images) {
			continue
		}

		if c.String("event") != "" && !stringInSlice(c.String("event"), secret.Events) {
			continue
		}

		tmpl.Execute(os.Stdout, secret)
	}

	return nil
}

// template for secret list items
var tmplOrgSecretList = "\x1b[33m{{ .Name }} \x1b[0m" + `
Images: {{ list .Images }}
Events: {{ list .Events }}
`

var orgSecretFuncMap = template.FuncMap{
	"list": func(s []string) string {
		return strings.Join(s, ", ")
	},
}
