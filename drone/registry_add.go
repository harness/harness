package main

import (
	"github.com/drone/drone/model"
	"github.com/urfave/cli"
)

var registryCreateCmd = cli.Command{
	Name:   "add",
	Usage:  "adds a registry",
	Action: registryCreate,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "repository",
			Usage: "repository name (e.g. octocat/hello-world)",
		},
		cli.StringFlag{
			Name:  "hostname",
			Usage: "registry hostname",
			Value: "docker.io",
		},
		cli.StringFlag{
			Name:  "username",
			Usage: "registry username",
		},
		cli.StringFlag{
			Name:  "password",
			Usage: "registry password",
		},
	},
}

func registryCreate(c *cli.Context) error {
	var (
		hostname = c.String("hostname")
		username = c.String("username")
		password = c.String("password")
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
	registry := &model.Registry{
		Address:  hostname,
		Username: username,
		Password: password,
	}
	_, err = client.RegistryCreate(owner, name, registry)
	if err != nil {
		return err
	}
	return nil
}
