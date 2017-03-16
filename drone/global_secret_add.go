package main

import "github.com/urfave/cli"

var globalSecretAddCmd = cli.Command{
	Name:      "add",
	Usage:     "adds a secret",
	ArgsUsage: "[key] [value]",
	Action:    globalSecretAdd,
	Flags:     secretAddFlags(),
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
