package main

import (
	"log"

	"github.com/codegangsta/cli"
)

var globalSecretRemoveCmd = cli.Command{
	Name:  "rm",
	Usage: "remove a secret",
	Action: func(c *cli.Context) {
		if err := globalSecretRemove(c); err != nil {
			log.Fatalln(err)
		}
	},
}

func globalSecretRemove(c *cli.Context) error {
	if len(c.Args()) != 1 {
		cli.ShowSubcommandHelp(c)
		return nil
	}

	secret := c.Args().First()

	client, err := newClient(c)
	if err != nil {
		return err
	}

	return client.GlobalSecretDel(secret)
}
