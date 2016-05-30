package main

import (
	"fmt"
	"log"
	"regexp"

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
		re := regexp.MustCompile("client error \\d{3}")
        err_ := re.FindAllString(err.Error(), -1)
        if err_ != nil {
            return fmt.Errorf("%s", err_[0])
        }
	}

	fmt.Printf("Successfully removed repository %s/%s\n", owner, name)
	return nil
}
