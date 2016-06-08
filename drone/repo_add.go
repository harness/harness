package main

import (
	"fmt"
	"log"

	"github.com/codegangsta/cli"
)

var repoAddCmd = cli.Command{
	Name:  "add",
	Usage: "add a repository",
	Action: func(c *cli.Context) {
		if err := repoAdd(c); err != nil {
			log.Fatalln(err)
		}
	},
}

func repoAdd(c *cli.Context) error {
	repo := c.Args().First()
	owner, name, err := parseRepo(repo)
	if err != nil {
		return err
	}

	client, err := newClient(c)
	if err != nil {
		return err
	}

	if _, err := client.RepoPost(owner, name); err != nil {
		return err
	}
	fmt.Printf("Successfully activated repository %s/%s\n", owner, name)
	return nil
}
