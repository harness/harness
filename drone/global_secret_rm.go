package main

import "github.com/urfave/cli"

var globalSecretRemoveCmd = cli.Command{
	Name:   "rm",
	Usage:  "remove a secret",
	Action: globalSecretRemove,
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
