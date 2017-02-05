package main

import (
	"log"

	"github.com/codegangsta/cli"
)

var globalSecretListCmd = cli.Command{
	Name:  "ls",
	Usage: "list all secrets",
	Action: func(c *cli.Context) {
		if err := globalSecretList(c); err != nil {
			log.Fatalln(err)
		}
	},
	Flags: secretListFlags(),
}

func globalSecretList(c *cli.Context) error {
	client, err := newClient(c)
	if err != nil {
		return err
	}

	secrets, err := client.GlobalSecretList()

	if err != nil || len(secrets) == 0 {
		return err
	}

	return secretDisplayList(secrets, c)
}
