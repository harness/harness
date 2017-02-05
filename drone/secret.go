package main

import (
	"io/ioutil"
	"os"
	"strings"
	"text/template"

	"github.com/codegangsta/cli"
	"github.com/drone/drone/model"
)

var secretCmd = cli.Command{
	Name:  "secret",
	Usage: "manage secrets",
	Subcommands: []cli.Command{
		secretAddCmd,
		secretRemoveCmd,
		secretListCmd,
	},
}

func secretAddFlags() []cli.Flag {
	return []cli.Flag{
		cli.StringSliceFlag{
			Name:  "event",
			Usage: "inject the secret for these event types",
			Value: &cli.StringSlice{
				model.EventPush,
				model.EventTag,
				model.EventDeploy,
			},
		},
		cli.StringSliceFlag{
			Name:  "image",
			Usage: "inject the secret for these image types",
			Value: &cli.StringSlice{},
		},
		cli.StringFlag{
			Name:  "input",
			Usage: "input secret value from a file",
		},
		cli.BoolFlag{
			Name:  "skip-verify",
			Usage: "skip verification for the secret",
		},
		cli.BoolFlag{
			Name:  "conceal",
			Usage: "conceal secret in build logs",
		},
	}
}

func secretListFlags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:  "format",
			Usage: "format output",
			Value: tmplSecretList,
		},
		cli.StringFlag{
			Name:  "image",
			Usage: "filter by image",
		},
		cli.StringFlag{
			Name:  "event",
			Usage: "filter by event",
		},
	}
}

func secretParseCmd(name string, value string, c *cli.Context) (*model.Secret, error) {
	secret := &model.Secret{}
	secret.Name = name
	secret.Value = value
	secret.Images = c.StringSlice("image")
	secret.Events = c.StringSlice("event")
	secret.SkipVerify = c.Bool("skip-verify")
	secret.Conceal = c.Bool("conceal")

	// TODO(bradrydzewski) below we use an @ sybmol to denote that the secret
	// value should be loaded from a file (inspired by curl). I'd prefer to use
	// a --input flag to explicitly specify a filepath instead.

	if strings.HasPrefix(secret.Value, "@") {
		path := secret.Value[1:]
		out, ferr := ioutil.ReadFile(path)
		if ferr != nil {
			return nil, ferr
		}
		secret.Value = string(out)
	}

	return secret, nil
}

func secretDisplayList(secrets []*model.Secret, c *cli.Context) error {
	tmpl, err := template.New("_").Funcs(secretFuncMap).Parse(c.String("format") + "\n")

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
var tmplSecretList = "\x1b[33m{{ .Name }} \x1b[0m" + `
Events: {{ list .Events }}
SkipVerify: {{ .SkipVerify }}
Conceal: {{ .Conceal }}
`

var secretFuncMap = template.FuncMap{
	"list": func(s []string) string {
		return strings.Join(s, ", ")
	},
}
