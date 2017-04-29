package info

import (
	"os"
	"text/template"

	"github.com/urfave/cli"

	"github.com/drone/drone/drone/internal"
)

// Command exports the info command.
var Command = cli.Command{
	Name:   "info",
	Usage:  "show information about the current user",
	Action: info,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:   "format",
			Usage:  "format output",
			Value:  tmplInfo,
			Hidden: true,
		},
	},
}

func info(c *cli.Context) error {
	client, err := internal.NewClient(c)
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
