package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/drone/drone/model"
)

var orgSecretAddCmd = cli.Command{
	Name:      "add",
	Usage:     "adds a secret",
	ArgsUsage: "[org] [key] [value]",
	Action: func(c *cli.Context) {
		if err := orgSecretAdd(c); err != nil {
			log.Fatalln(err)
		}
	},
	Flags: []cli.Flag{
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
	},
}

func orgSecretAdd(c *cli.Context) error {
	if len(c.Args()) != 3 {
		cli.ShowSubcommandHelp(c)
		return nil
	}

	team := c.Args().First()
	name := c.Args().Get(1)
	value := c.Args().Get(2)

	secret := &model.Secret{}
	secret.Name = name
	secret.Value = value
	secret.Images = c.StringSlice("image")
	secret.Events = c.StringSlice("event")

	if len(secret.Images) == 0 {
		return fmt.Errorf("Please specify the --image parameter")
	}

	// TODO(bradrydzewski) below we use an @ sybmol to denote that the secret
	// value should be loaded from a file (inspired by curl). I'd prefer to use
	// a --input flag to explicitly specify a filepath instead.

	if strings.HasPrefix(secret.Value, "@") {
		path := secret.Value[1:]
		out, ferr := ioutil.ReadFile(path)
		if ferr != nil {
			return ferr
		}
		secret.Value = string(out)
	}

	client, err := newClient(c)
	if err != nil {
		return err
	}

	return client.TeamSecretPost(team, secret)
}
