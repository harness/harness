package registry

import (
	"html/template"
	"os"

	"github.com/urfave/cli"

	"github.com/drone/drone/drone/internal"
)

var registryListCmd = cli.Command{
	Name:   "ls",
	Usage:  "list regitries",
	Action: registryList,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "repository",
			Usage: "repository name (e.g. octocat/hello-world)",
		},
		cli.StringFlag{
			Name:   "format",
			Usage:  "format output",
			Value:  tmplRegistryList,
			Hidden: true,
		},
	},
}

func registryList(c *cli.Context) error {
	var (
		format   = c.String("format") + "\n"
		reponame = c.String("repository")
	)
	if reponame == "" {
		reponame = c.Args().First()
	}
	owner, name, err := internal.ParseRepo(reponame)
	if err != nil {
		return err
	}
	client, err := internal.NewClient(c)
	if err != nil {
		return err
	}
	list, err := client.RegistryList(owner, name)
	if err != nil {
		return err
	}
	tmpl, err := template.New("_").Parse(format)
	if err != nil {
		return err
	}
	for _, registry := range list {
		tmpl.Execute(os.Stdout, registry)
	}
	return nil
}

// template for build list information
var tmplRegistryList = "\x1b[33m{{ .Address }} \x1b[0m" + `
Username: {{ .Username }}
Email: {{ .Email }}
`
