package main

import (
	"fmt"

	"github.com/codegangsta/cli"
	"github.com/drone/drone/shared/model"
)

// NewWhoamiCommand returns the CLI command for "whoami".
func NewWhoamiCommand() cli.Command {
	return cli.Command{
		Name:  "whoami",
		Usage: "outputs the current user",
		Flags: []cli.Flag{},
		Action: func(c *cli.Context) {
			handle(c, whoamiCommandFunc)
		},
	}
}

// whoamiCommandFunc executes the "logout" command.
func whoamiCommandFunc(c *cli.Context, client *Client) error {
	user := model.User{}
	err := client.Do("GET", "/v1/user", nil, &user)
	if err != nil {
		return err
	}

	fmt.Println(user.Login)
	return nil
}
