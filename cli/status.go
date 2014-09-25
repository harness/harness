package main

import (
	"fmt"

	"github.com/codegangsta/cli"
	"github.com/drone/drone/client"
)

// NewStatusCommand returns the CLI command for "status".
func NewStatusCommand() cli.Command {
	return cli.Command{
		Name:  "status",
		Usage: "display a repository build status",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "b, branch",
				Usage: "branch to display",
			},
		},
		Action: func(c *cli.Context) {
			handle(c, statusCommandFunc)
		},
	}
}

// statusCommandFunc executes the "status" command.
func statusCommandFunc(c *cli.Context, client *client.Client) error {
	var host, owner, repo, branch string
	var args = c.Args()

	if len(args) != 0 {
		host, owner, repo = parseRepo(args[0])
	}

	if c.IsSet("branch") {
		branch = c.String("branch")
	} else {
		branch = "master"
	}

	builds, err := client.Commits.ListBranch(host, owner, repo, branch)
	if err != nil {
		return err
	} else if len(builds) == 0 {
		return nil
	}

	var build = builds[len(builds)-1]
	fmt.Printf("%s\t%s\t%s\t%s\t%v", build.Status, build.ShaShort(), build.Timestamp, build.Author, build.Message)
	return nil
}
