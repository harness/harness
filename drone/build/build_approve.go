package build

import (
	"fmt"
	"strconv"

	"github.com/drone/drone/drone/internal"
	"github.com/urfave/cli"
)

var buildApproveCmd = cli.Command{
	Name:   "approve",
	Usage:  "approve a build",
	Action: buildApprove,
}

func buildApprove(c *cli.Context) (err error) {
	repo := c.Args().First()
	owner, name, err := internal.ParseRepo(repo)
	if err != nil {
		return err
	}
	number, err := strconv.Atoi(c.Args().Get(1))
	if err != nil {
		return err
	}

	client, err := internal.NewClient(c)
	if err != nil {
		return err
	}

	_, err = client.BuildApprove(owner, name, number)
	if err != nil {
		return err
	}

	fmt.Printf("Approving build %s/%s#%d\n", owner, name, number)
	return nil
}
