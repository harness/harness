package main

import (
	"log"

	"github.com/codegangsta/cli"
)

var secretRemoveCmd = cli.Command{
	Name:  "rm",
	Usage: "remove a secret",
	Action: func(c *cli.Context) {
		if err := secretRemove(c); err != nil {
			log.Fatalln(err)
		}
	},
}

func secretRemove(c *cli.Context) error {
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
