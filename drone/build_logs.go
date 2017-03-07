package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/codegangsta/cli"
	"github.com/drone/drone/build"
)

var buildLogsCmd = cli.Command{
	Name:  "logs",
	Usage: "show build logs",
	Action: func(c *cli.Context) {
		if err := buildLogs(c); err != nil {
			log.Fatalln(err)
		}
	},
}

func buildLogs(c *cli.Context) error {
	repo := c.Args().First()
	owner, name, err := parseRepo(repo)
	if err != nil {
		return err
	}

	client, err := newClient(c)
	if err != nil {
		return err
	}

	buildArg := c.Args().Get(1)
	var number int
	if buildArg == "last" {
		// Fetch the build number from the last build
		build, err := client.BuildLast(owner, name, "")
		if err != nil {
			return err
		}
		number = build.Number
	} else {
		number, err = strconv.Atoi(buildArg)
		if err != nil {
			return err
		}
	}

	job, _ := strconv.Atoi(c.Args().Get(2))
	if job == 0 {
		job = 1
	}

	r, err := client.BuildLogs(owner, name, number, job)
	if err != nil {
		return err
	}
	defer r.Close()

	dec := json.NewDecoder(r)
	fmt.Printf("Logs for build %s/%s#%d.%d\n", owner, name, number, job)
	var line build.Line

	_, err = dec.Token()
	if err != nil {
		return err
	}

	for dec.More() {
		if err = dec.Decode(&line); err != nil {
			return err
		}
		fmt.Printf("%s\n", line.Out)
	}

	_, err = dec.Token()
	if err != nil {
		return err
	}

	return nil
}
