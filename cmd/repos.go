package main

import (
	"fmt"

	"github.com/codegangsta/cli"
	"github.com/drone/drone/client"
)

// NewReposCommand returns the CLI command for "repos".
func NewReposCommand() cli.Command {
	return cli.Command{
		Name:  "repos",
		Usage: "lists active remote repositories",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "a, all",
				Usage: "list all repositories",
			},
		},
		Action: func(c *cli.Context) {
			handle(c, reposCommandFunc)
		},
	}
}

// reposCommandFunc executes the "repos" command.
func reposCommandFunc(c *cli.Context, client *client.Client) error {
	repos, err := client.Repos.List()
	if err != nil {
		return err
	}

	var all = c.Bool("a")
	for _, repo := range repos {
		if !all && !repo.Active {
			continue
		}

		fmt.Printf("%s/%s/%s\n", repo.Host, repo.Owner, repo.Name)
	}
	return nil
}
