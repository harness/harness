package main

import (
	"fmt"
	"html/template"
	"os"
	"strconv"

	"github.com/drone/drone/model"
	"github.com/urfave/cli"
)

var deployCmd = cli.Command{
	Name:   "deploy",
	Usage:  "deploy code",
	Action: deploy,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "format",
			Usage: "format output",
			Value: tmplDeployInfo,
		},
		cli.StringFlag{
			Name:  "branch",
			Usage: "branch filter",
			Value: "master",
		},
		cli.StringFlag{
			Name:  "event",
			Usage: "event filter",
			Value: model.EventPush,
		},
		cli.StringFlag{
			Name:  "status",
			Usage: "status filter",
			Value: model.StatusSuccess,
		},
		cli.StringSliceFlag{
			Name:  "param, p",
			Usage: "custom parameters to be injected into the job environment. Format: KEY=value",
		},
	},
}

func deploy(c *cli.Context) error {
	repo := c.Args().First()
	owner, name, err := parseRepo(repo)
	if err != nil {
		return err
	}

	client, err := newClient(c)
	if err != nil {
		return err
	}

	branch := c.String("branch")
	event := c.String("event")
	status := c.String("status")

	buildArg := c.Args().Get(1)
	var number int
	if buildArg == "last" {
		// Fetch the build number from the last build
		builds, err := client.BuildList(owner, name)
		if err != nil {
			return err
		}
		for _, build := range builds {
			if branch != "" && build.Branch != branch {
				continue
			}
			if event != "" && build.Event != event {
				continue
			}
			if status != "" && build.Status != status {
				continue
			}
			if build.Number > number {
				number = build.Number
			}
		}
		if number == 0 {
			return fmt.Errorf("Cannot deploy failure build")
		}
	} else {
		number, err = strconv.Atoi(buildArg)
		if err != nil {
			return err
		}
	}

	env := c.Args().Get(2)
	if env == "" {
		return fmt.Errorf("Please specify the target environment (ie production)")
	}

	params := parseKVPairs(c.StringSlice("param"))

	deploy, err := client.Deploy(owner, name, number, env, params)
	if err != nil {
		return err
	}

	tmpl, err := template.New("_").Parse(c.String("format"))
	if err != nil {
		return err
	}
	return tmpl.Execute(os.Stdout, deploy)
}

// template for deployment information
var tmplDeployInfo = `Number: {{ .Number }}
Status: {{ .Status }}
Commit: {{ .Commit }}
Branch: {{ .Branch }}
Ref: {{ .Ref }}
Message: {{ .Message }}
Author: {{ .Author }}
Target: {{ .Deploy }}
`
