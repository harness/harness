package build

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/cncd/pipeline/pipeline/rpc"
	"github.com/drone/drone/drone/internal"
	"github.com/urfave/cli"
)

var buildLogsCmd = cli.Command{
	Name:   "logs",
	Usage:  "show build logs",
	Action: buildLogsDisabled,
}

func buildLogsDisabled(c *cli.Context) error {
	return fmt.Errorf("Command temporarily disabled. See https://github.com/drone/drone/issues/2005")
}

func buildLogs(c *cli.Context) error {
	repo := c.Args().First()
	owner, name, err := internal.ParseRepo(repo)
	if err != nil {
		return err
	}

	client, err := internal.NewClient(c)
	if err != nil {
		return err
	}

	buildArg := c.Args().Get(1)
	var number int
	if buildArg == "" {
		// Fetch the build number from the last build
		build, err := client.BuildLast(owner, name, "")
		if err != nil {
			return err
		}
		number = build.Number
	} else {
		number, err = strconv.Atoi(buildArg)
		if err != nil {
			return fmt.Errorf("Error: Invalid number or missing job number. eg 100")
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
	var line rpc.Line

	_, err = dec.Token()
	if err != nil {
		return err
	}

	for dec.More() {
		if err = dec.Decode(&line); err != nil {
			return err
		}
		fmt.Printf("%s", line.Out)
	}

	_, err = dec.Token()
	if err != nil {
		return err
	}

	return nil
}
