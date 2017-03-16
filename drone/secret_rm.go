package main

import "github.com/urfave/cli"

var secretRemoveCmd = cli.Command{
	Name:   "rm",
	Usage:  "remove a secret",
	Action: secretRemove,
}

func secretRemove(c *cli.Context) error {
	repo := c.Args().First()
	owner, name, err := parseRepo(repo)
	if err != nil {
		return err
	}

	secret := c.Args().Get(1)

	client, err := newClient(c)
	if err != nil {
		return err
	}
	return client.SecretDel(owner, name, secret)
}
