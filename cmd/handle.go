package main

import (
	"os"

	"github.com/codegangsta/cli"
)

type handlerFunc func(*cli.Context, *Client) error

// handle wraps the command function handlers and
// sets up the environment.
func handle(c *cli.Context, fn handlerFunc) {
	client := Client{}
	client.Token = os.Getenv("DRONE_TOKEN")
	client.URL = os.Getenv("DRONE_HOST")

	// if no url is provided we can default
	// to the hosted Drone service.
	if len(client.URL) == 0 {
		client.URL = "http://test.drone.io"
	}

	// handle the function
	if err := fn(c, &client); err != nil {
		println(err.Error())
		os.Exit(1)
	}
}
