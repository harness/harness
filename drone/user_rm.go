package main

import (
	"fmt"
	"log"

	"github.com/codegangsta/cli"
)

var userRemoveCmd = cli.Command{
	Name:  "rm",
	Usage: "remove a user",
	Action: func(c *cli.Context) {
		if err := userRemove(c); err != nil {
			log.Fatalln(err)
		}
	},
}

func userRemove(c *cli.Context) error {
	login := c.Args().First()

	client, err := newClient(c)
	if err != nil {
		return err
	}

	if err := client.UserDel(login); err != nil {
		return err
	}
	fmt.Printf("Successfully removed user %s\n", login)
	return nil
}
