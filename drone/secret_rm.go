package main

import "github.com/urfave/cli"

var secretDeleteCmd = cli.Command{
	Name:   "rm",
	Usage:  "remove a secret",
	Action: secretDelete,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "repository",
			Usage: "repository name (e.g. octocat/hello-world)",
		},
		cli.StringFlag{
			Name:  "name",
			Usage: "secret name",
		},
	},
}

func secretDelete(c *cli.Context) error {
	var (
		secret   = c.String("name")
		reponame = c.String("repository")
	)
	if reponame == "" {
		reponame = c.Args().First()
	}
	owner, name, err := parseRepo(reponame)
	if err != nil {
		return err
	}
	client, err := newClient(c)
	if err != nil {
		return err
	}
	return client.SecretDelete(owner, name, secret)
}
