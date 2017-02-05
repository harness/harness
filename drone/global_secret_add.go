package main

import (
	"log"

	"github.com/codegangsta/cli"
)

var globalSecretAddCmd = cli.Command{
	Name:      "add",
	Usage:     "adds a secret",
	ArgsUsage: "[key] [value]",
	Action: func(c *cli.Context) {
		if err := globalSecretAdd(c); err != nil {
			log.Fatalln(err)
		}
	},
	Flags: secretAddFlags(),
}

func globalSecretAdd(c *cli.Context) error {
	if len(c.Args()) != 2 {
		cli.ShowSubcommandHelp(c)
		return nil
	}

	name := c.Args().First()
	value := c.Args().Get(1)

	secret, err := secretParseCmd(name, value, c)
	if err != nil {
		return err
	}

	client, err := newClient(c)
	if err != nil {
		return err
	}

	return client.GlobalSecretPost(secret)
}
