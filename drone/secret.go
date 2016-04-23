package main

import (
	"fmt"
	"log"

	"github.com/drone/drone/model"

	"github.com/codegangsta/cli"
)

// SecretCmd is the exported command for managing secrets.
var SecretCmd = cli.Command{
	Name:  "secret",
	Usage: "manage secrets",
	Subcommands: []cli.Command{
		// Secret Add
		{
			Name:      "add",
			Usage:     "add a secret",
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
			},
		},
		// Secret Delete
		{
			Name:  "rm",
			Usage: "remove a secret",
			Action: func(c *cli.Context) {
				if err := secretDel(c); err != nil {
					log.Fatalln(err)
				}
			},
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

	client, err := newClient(c)
	if err != nil {
		return err
	}

	return client.SecretPost(owner, name, secret)
}

func secretDel(c *cli.Context) error {
	repo := c.Args().First()
	owner, name, err := parseRepo(repo)
	if err != nil {
		return err
	}

	secret := c.Args().Get(1)

	client, err := newClient(c)
	if err != nil {
		return err
	}
	return client.SecretDel(owner, name, secret)
}
