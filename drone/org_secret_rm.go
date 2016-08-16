package main

import (
	"log"

	"github.com/codegangsta/cli"
)

var orgSecretRemoveCmd = cli.Command{
	Name:  "rm",
	Usage: "remove a secret",
	Action: func(c *cli.Context) {
		if err := orgSecretRemove(c); err != nil {
			log.Fatalln(err)
		}
	},
}

func orgSecretRemove(c *cli.Context) error {
	if len(c.Args()) != 2 {
		cli.ShowSubcommandHelp(c)
		return nil
	}

	team := c.Args().First()
	secret := c.Args().Get(1)

	client, err := newClient(c)
	if err != nil {
		return err
	}

	return client.TeamSecretDel(team, secret)
}
