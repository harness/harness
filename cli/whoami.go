package main

import (
	"fmt"

	"github.com/codegangsta/cli"
	"github.com/drone/drone/client"
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

// whoamiCommandFunc communicates with the server and echoes
// the currently authenticated user.
func whoamiCommandFunc(c *cli.Context, client *client.Client) error {
	user, err := client.Users.GetCurrent()
	if err != nil {
		return err
	}

	fmt.Println(user.Login)
	return nil
}
