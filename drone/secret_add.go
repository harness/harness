package main

import (
	"log"

	"github.com/codegangsta/cli"
)

var secretAddCmd = cli.Command{
	Name:      "add",
	Usage:     "adds a secret",
	ArgsUsage: "[repo] [key] [value]",
	Action: func(c *cli.Context) {
		if err := secretAdd(c); err != nil {
			log.Fatalln(err)
		}
	},
	Flags: secretAddFlags(),
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

	secret, err := secretParseCmd(tail[0], tail[1], c)
	if err != nil {
		return err
	}

	client, err := newClient(c)
	if err != nil {
		return err
	}

	return client.SecretPost(owner, name, secret)
}
