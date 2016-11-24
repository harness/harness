package main

import (
	"fmt"
	"log"

	"github.com/codegangsta/cli"
)

var repoChownCmd = cli.Command{
	Name:  "chown",
	Usage: "assume ownership of a repository",
	Action: func(c *cli.Context) {
		if err := repoChown(c); err != nil {
			log.Fatalln(err)
		}
	},
}

func repoChown(c *cli.Context) error {
	repo := c.Args().First()
	owner, name, err := parseRepo(repo)
	if err != nil {
		return err
	}

	client, err := newClient(c)
	if err != nil {
		return err
	}

	if _, err := client.RepoChown(owner, name); err != nil {
		return err
	}
	fmt.Printf("Successfully assumed ownership of repository %s/%s\n", owner, name)
	return nil
}
