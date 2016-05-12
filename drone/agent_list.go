package main

import (
	"html/template"
	"log"
	"os"
	"time"

	"github.com/codegangsta/cli"
)

var agentsCmd = cli.Command{
	Name:  "agents",
	Usage: "manage agents",
	Action: func(c *cli.Context) {
		if err := agentList(c); err != nil {
			log.Fatalln(err)
		}
	},
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "format",
			Usage: "format output",
			Value: tmplAgentList,
		},
	},
}

func agentList(c *cli.Context) error {
	client, err := newClient(c)
	if err != nil {
		return err
	}

	agents, err := client.AgentList()
	if err != nil {
		return err
	}

	tmpl, err := template.New("_").Funcs(funcMap).Parse(c.String("format") + "\n")
	if err != nil {
		return err
	}

	for _, agent := range agents {
		tmpl.Execute(os.Stdout, agent)
	}
	return nil
}

// template for build list information
var tmplAgentList = "\x1b[33m{{ .Address }} \x1b[0m" + `
Platform: {{ .Platform }}
Capacity: {{ .Capacity }} concurrent build(s)
Pinged: {{ since .Updated }} ago
Uptime: {{ since .Created }}
`

var funcMap = template.FuncMap{
	"since": func(t int64) string {
		d := time.Now().Sub(time.Unix(t, 0))
		return d.String()
	},
}
