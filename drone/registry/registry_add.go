package registry

import (
	"io/ioutil"
	"strings"

	"github.com/drone/drone/drone/internal"
	"github.com/drone/drone/model"

	"github.com/urfave/cli"
)

var registryCreateCmd = cli.Command{
	Name:   "add",
	Usage:  "adds a registry",
	Action: registryCreate,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "repository",
			Usage: "repository name (e.g. octocat/hello-world)",
		},
		cli.StringFlag{
			Name:  "hostname",
			Usage: "registry hostname",
			Value: "docker.io",
		},
		cli.StringFlag{
			Name:  "username",
			Usage: "registry username",
		},
		cli.StringFlag{
			Name:  "password",
			Usage: "registry password",
		},
	},
}

func registryCreate(c *cli.Context) error {
	var (
		hostname = c.String("hostname")
		username = c.String("username")
		password = c.String("password")
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
	registry := &model.Registry{
		Address:  hostname,
		Username: username,
		Password: password,
	}
	if strings.HasPrefix(registry.Password, "@") {
		path := strings.TrimPrefix(registry.Password, "@")
		out, ferr := ioutil.ReadFile(path)
		if ferr != nil {
			return ferr
		}
		registry.Password = string(out)
	}
	_, err = client.RegistryCreate(owner, name, registry)
	if err != nil {
		return err
	}
	return nil
}
