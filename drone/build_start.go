package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/codegangsta/cli"
	"github.com/drone/drone/model"
)

var buildStartCmd = cli.Command{
	Name:  "start",
	Usage: "start a build",
	Action: func(c *cli.Context) {
		if err := buildStart(c); err != nil {
			log.Fatalln(err)
		}
	},
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "fork",
			Usage: "fork the build",
		},
		cli.StringSliceFlag{
			Name:  "param, p",
			Usage: "custom parameters to be injected into the job environment. Format: KEY=value",
		},
	},
}

func buildStart(c *cli.Context) (err error) {
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

	params := parseKVPairs(c.StringSlice("param"))

	var build *model.Build
	if c.Bool("fork") {
		build, err = client.BuildFork(owner, name, number, params)
	} else {
		build, err = client.BuildStart(owner, name, number, params)
	}
	if err != nil {
		return err
	}

	fmt.Printf("Starting build %s/%s#%d\n", owner, name, build.Number)
	return nil
}
