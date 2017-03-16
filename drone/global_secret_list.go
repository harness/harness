package main

import "github.com/urfave/cli"

var globalSecretListCmd = cli.Command{
	Name:   "ls",
	Usage:  "list all secrets",
	Action: globalSecretList,
	Flags:  secretListFlags(),
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
