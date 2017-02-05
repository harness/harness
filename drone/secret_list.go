package main

import (
	"log"

	"github.com/codegangsta/cli"
)

var secretListCmd = cli.Command{
	Name:  "ls",
	Usage: "list all secrets",
	Action: func(c *cli.Context) {
		if err := secretList(c); err != nil {
			log.Fatalln(err)
		}
	},
	Flags: secretListFlags(),
}

func secretList(c *cli.Context) error {
	owner, name, err := parseRepo(c.Args().First())

	if err != nil {
		return err
	}

	client, err := newClient(c)

	if err != nil {
		return err
	}

	secrets, err := client.SecretList(owner, name)

	if err != nil || len(secrets) == 0 {
		return err
	}

	return secretDisplayList(secrets, c)
}
