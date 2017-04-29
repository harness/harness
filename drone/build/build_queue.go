package build

import (
	"fmt"
	"os"
	"text/template"

	"github.com/drone/drone/drone/internal"
	"github.com/urfave/cli"
)

var buildQueueCmd = cli.Command{
	Name:   "queue",
	Usage:  "show build queue",
	Action: buildQueue,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "format",
			Usage: "format output",
			Value: tmplBuildQueue,
		},
	},
}

func buildQueue(c *cli.Context) error {

	client, err := internal.NewClient(c)
	if err != nil {
		return err
	}

	builds, err := client.BuildQueue()
	if err != nil {
		return err
	}

	if len(builds) == 0 {
		fmt.Println("there are no pending or running builds")
		return nil
	}

	tmpl, err := template.New("_").Parse(c.String("format") + "\n")
	if err != nil {
		return err
	}

	for _, build := range builds {
		tmpl.Execute(os.Stdout, build)
	}
	return nil
}

// template for build list information
var tmplBuildQueue = "\x1b[33m{{ .FullName }} #{{ .Number }} \x1b[0m" + `
Status: {{ .Status }}
Event: {{ .Event }}
Commit: {{ .Commit }}
Branch: {{ .Branch }}
Ref: {{ .Ref }}
Author: {{ .Author }} {{ if .Email }}<{{.Email}}>{{ end }}
Message: {{ .Message }}
`
