package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/drone/drone/model"
)

var secretAddCmd = cli.Command{
	Name:      "add",
	Usage:     "adds a secret",
	ArgsUsage: "[repo] [key] [value]",
	Action: func(c *cli.Context) {
		if err := secretAdd(c); err != nil {
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

func secretAdd(c *cli.Context) error {
	repo := c.Args().First()
	owner, name, err := parseRepo(repo)
	if err != nil {
		return err
	}

	tail := c.Args().Tail()
	if len(tail) != 2 {
		cli.ShowSubcommandHelp(c)
		return nil
	}

	secret := &model.Secret{}
	secret.Name = tail[0]
	secret.Value = tail[1]
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

	return client.SecretPost(owner, name, secret)
}
