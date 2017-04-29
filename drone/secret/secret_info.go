package secret

import (
	"html/template"
	"os"

	"github.com/urfave/cli"

	"github.com/drone/drone/drone/internal"
)

var secretInfoCmd = cli.Command{
	Name:   "info",
	Usage:  "display secret info",
	Action: secretInfo,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "repository",
			Usage: "repository name (e.g. octocat/hello-world)",
		},
		cli.StringFlag{
			Name:  "name",
			Usage: "secret name",
		},
		cli.StringFlag{
			Name:   "format",
			Usage:  "format output",
			Value:  tmplSecretList,
			Hidden: true,
		},
	},
}

func secretInfo(c *cli.Context) error {
	var (
		secretName = c.String("name")
		repoName   = c.String("repository")
		format     = c.String("format") + "\n"
	)
	if repoName == "" {
		repoName = c.Args().First()
	}
	owner, name, err := internal.ParseRepo(repoName)
	if err != nil {
		return err
	}
	client, err := internal.NewClient(c)
	if err != nil {
		return err
	}
	secret, err := client.Secret(owner, name, secretName)
	if err != nil {
		return err
	}
	tmpl, err := template.New("_").Funcs(secretFuncMap).Parse(format)
	if err != nil {
		return err
	}
	return tmpl.Execute(os.Stdout, secret)
}
