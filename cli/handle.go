package main

import (
	"os"

	"github.com/codegangsta/cli"
	"github.com/drone/drone/client"
)

type handlerFunc func(*cli.Context, *client.Client) error

// handle wraps the command function handlers and
// sets up the environment.
func handle(c *cli.Context, fn handlerFunc) {
	var token = c.GlobalString("token")
	var server = c.GlobalString("server")

	// if no server url is provided we can default
	// to the hosted Drone service.
	if len(server) == 0 {
		server = "http://test.drone.io"
	}

	// create the drone client
	client := client.New(token, server)

	// handle the function
	if err := fn(c, client); err != nil {
		println(err.Error())
		os.Exit(1)
	}
}
