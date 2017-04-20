package main

import (
	"io/ioutil"
	"strings"

	"github.com/drone/drone/model"
	"github.com/urfave/cli"
)

var secretCreateCmd = cli.Command{
	Name:   "add",
	Usage:  "adds a secret",
	Action: secretCreate,
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

func secretCreate(c *cli.Context) error {
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
		Events: c.StringSlice("event"),
	}
	if len(secret.Events) == 0 {
		secret.Events = defaultSecretEvents
	}
	if strings.HasPrefix(secret.Value, "@") {
		path := strings.TrimPrefix(secret.Value, "@")
		out, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		secret.Value = string(out)
	}
	_, err = client.SecretCreate(owner, name, secret)
	return err
}

var defaultSecretEvents = []string{
	model.EventPush,
	model.EventTag,
	model.EventDeploy,
}
