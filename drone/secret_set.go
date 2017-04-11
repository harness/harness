package main

import (
	"github.com/drone/drone/model"
	"github.com/urfave/cli"
)

var secretUpdateCmd = cli.Command{
	Name:   "update",
	Usage:  "update a secret",
	Action: secretUpdate,
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
			Name:  "value",
			Usage: "secret value",
		},
		cli.StringSliceFlag{
			Name:  "event",
			Usage: "secret limited to these events",
		},
		cli.StringSliceFlag{
			Name:  "image",
			Usage: "secret limited to these images",
		},
	},
}

func secretUpdate(c *cli.Context) error {
	reponame := c.String("repository")
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
	secret := &model.Secret{
		Name:   c.String("name"),
		Value:  c.String("value"),
		Images: c.StringSlice("image"),
		Events: c.StringSlice("events"),
	}
	_, err = client.SecretUpdate(owner, name, secret)
	return err
}
