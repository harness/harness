package main

import (
	"log"

	"github.com/codegangsta/cli"
)

var orgSecretListCmd = cli.Command{
	Name:  "ls",
	Usage: "list all secrets",
	Action: func(c *cli.Context) {
		if err := orgSecretList(c); err != nil {
			log.Fatalln(err)
		}
	},
	Flags: secretListFlags(),
}

func orgSecretList(c *cli.Context) error {
	if len(c.Args()) != 1 {
		cli.ShowSubcommandHelp(c)
		return nil
	}

	team := c.Args().First()

	client, err := newClient(c)
	if err != nil {
		return err
	}

	secrets, err := client.TeamSecretList(team)

	if err != nil || len(secrets) == 0 {
		return err
	}

	return secretDisplayList(secrets, c)
}
