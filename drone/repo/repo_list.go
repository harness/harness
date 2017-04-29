package repo

import (
	"os"
	"text/template"

	"github.com/drone/drone/drone/internal"
	"github.com/urfave/cli"
)

var repoListCmd = cli.Command{
	Name:   "ls",
	Usage:  "list all repos",
	Action: repoList,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "format",
			Usage: "format output",
			Value: tmplRepoList,
		},
		cli.StringFlag{
			Name:  "org",
			Usage: "filter by organization",
		},
	},
}

func repoList(c *cli.Context) error {
	client, err := internal.NewClient(c)
	if err != nil {
		return err
	}

	repos, err := client.RepoList()
	if err != nil || len(repos) == 0 {
		return err
	}

	tmpl, err := template.New("_").Parse(c.String("format") + "\n")
	if err != nil {
		return err
	}

	org := c.String("org")
	for _, repo := range repos {
		if org != "" && org != repo.Owner {
			continue
		}
		tmpl.Execute(os.Stdout, repo)
	}
	return nil
}

// template for repository list items
var tmplRepoList = `{{ .FullName }}`
