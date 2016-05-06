package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/codegangsta/cli"
)

var buildStopCmd = cli.Command{
	Name:  "stop",
	Usage: "stop a build",
	Action: func(c *cli.Context) {
		if err := buildStop(c); err != nil {
			log.Fatalln(err)
		}
	},
}

func buildStop(c *cli.Context) (err error) {
	repo := c.Args().First()
	owner, name, err := parseRepo(repo)
	if err != nil {
		return err
	}
	number, err := strconv.Atoi(c.Args().Get(1))
	if err != nil {
		return err
	}
	job, _ := strconv.Atoi(c.Args().Get(2))
	if job == 0 {
		job = 1
	}

	client, err := newClient(c)
	if err != nil {
		return err
	}

	err = client.BuildStop(owner, name, number, job)
	if err != nil {
		return err
	}

	fmt.Printf("Stopping build %s/%s#%d.%d\n", owner, name, number, job)
	return nil
}
