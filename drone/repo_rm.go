package main

import (
	"fmt"
	"log"

	"github.com/codegangsta/cli"
)

var repoRemoveCmd = cli.Command{
	Name:  "rm",
	Usage: "remove a repository",
	Action: func(c *cli.Context) {
		if err := repoRemove(c); err != nil {
			log.Fatalln(err)
		}
	},
}

func repoRemove(c *cli.Context) error {
	repo := c.Args().First()
	owner, name, err := parseRepo(repo)
	if err != nil {
		return err
	}

	client, err := newClient(c)
	if err != nil {
		return err
	}

	if err := client.RepoDel(owner, name); err != nil {
		return err
	}
	fmt.Printf("Successfully removed repository %s/%s\n", owner, name)
	return nil
}
