package secret

import (
	"github.com/urfave/cli"

	"github.com/drone/drone/drone/internal"
)

var secretDeleteCmd = cli.Command{
	Name:   "rm",
	Usage:  "remove a secret",
	Action: secretDelete,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "repository",
			Usage: "repository name (e.g. octocat/hello-world)",
		},
		cli.StringFlag{
			Name:  "name",
			Usage: "secret name",
		},
	},
}

func secretDelete(c *cli.Context) error {
	var (
		secret   = c.String("name")
		reponame = c.String("repository")
	)
	if reponame == "" {
		reponame = c.Args().First()
	}
	owner, name, err := internal.ParseRepo(reponame)
	if err != nil {
		return err
	}
	client, err := internal.NewClient(c)
	if err != nil {
		return err
	}
	return client.SecretDelete(owner, name, secret)
}
