package main

import (
	"fmt"
	"log"

	"github.com/codegangsta/cli"
	"github.com/drone/drone/model"
)

var userAddCmd = cli.Command{
	Name:  "add",
	Usage: "adds a user",
	Action: func(c *cli.Context) {
		if err := userAdd(c); err != nil {
			log.Fatalln(err)
		}
	},
}

func userAdd(c *cli.Context) error {
	login := c.Args().First()

	client, err := newClient(c)
	if err != nil {
		return err
	}

	user, err := client.UserPost(&model.User{Login: login})
	if err != nil {
		return err
	}
	fmt.Printf("Successfully added user %s\n", user.Login)
	return nil
}
