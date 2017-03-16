package main

import "github.com/urfave/cli"

var orgSecretListCmd = cli.Command{
	Name:   "ls",
	Usage:  "list all secrets",
	Action: orgSecretList,
	Flags:  secretListFlags(),
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
