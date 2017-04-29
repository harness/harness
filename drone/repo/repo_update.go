package repo

import (
	"fmt"
	"time"

	"github.com/drone/drone/drone/internal"
	"github.com/drone/drone/model"
	"github.com/urfave/cli"
)

var repoUpdateCmd = cli.Command{
	Name:   "update",
	Usage:  "update a repository",
	Action: repoUpdate,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "trusted",
			Usage: "repository is trusted",
		},
		cli.BoolFlag{
			Name:  "gated",
			Usage: "repository is gated",
		},
		cli.DurationFlag{
			Name:  "timeout",
			Usage: "repository timeout",
		},
		cli.StringFlag{
			Name:  "config",
			Usage: "repository configuration path (e.g. .drone.yml)",
		},
	},
}

func repoUpdate(c *cli.Context) error {
	repo := c.Args().First()
	owner, name, err := internal.ParseRepo(repo)
	if err != nil {
		return err
	}

	client, err := internal.NewClient(c)
	if err != nil {
		return err
	}

	var (
		config  = c.String("config")
		timeout = c.Duration("timeout")
		trusted = c.Bool("trusted")
		gated   = c.Bool("gated")
	)

	patch := new(model.RepoPatch)
	if c.IsSet("trusted") {
		patch.IsTrusted = &trusted
	}
	if c.IsSet("gated") {
		patch.IsGated = &gated
	}
	if c.IsSet("timeout") {
		v := int64(timeout / time.Minute)
		patch.Timeout = &v
	}
	if c.IsSet("config") {
		patch.Config = &config
	}

	if _, err := client.RepoPatch(owner, name, patch); err != nil {
		return err
	}
	fmt.Printf("Successfully updated repository %s/%s\n", owner, name)
	return nil
}
