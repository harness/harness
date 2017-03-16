package main

import "github.com/urfave/cli"

var orgSecretAddCmd = cli.Command{
	Name:      "add",
	Usage:     "adds a secret",
	ArgsUsage: "[org] [key] [value]",
	Action:    orgSecretAdd,
	Flags:     secretAddFlags(),
}

func orgSecretAdd(c *cli.Context) error {
	if len(c.Args()) != 3 {
		cli.ShowSubcommandHelp(c)
		return nil
	}

	team := c.Args().First()
	name := c.Args().Get(1)
	value := c.Args().Get(2)

	secret, err := secretParseCmd(name, value, c)
	if err != nil {
		return err
	}

	client, err := newClient(c)
	if err != nil {
		return err
	}

	return client.TeamSecretPost(team, secret)
}
