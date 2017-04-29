package user

import (
	"fmt"
	"os"
	"text/template"

	"github.com/urfave/cli"

	"github.com/drone/drone/drone/internal"
)

var userInfoCmd = cli.Command{
	Name:   "info",
	Usage:  "show user details",
	Action: userInfo,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "format",
			Usage: "format output",
			Value: tmplUserInfo,
		},
	},
}

func userInfo(c *cli.Context) error {
	client, err := internal.NewClient(c)
	if err != nil {
		return err
	}

	login := c.Args().First()
	if len(login) == 0 {
		return fmt.Errorf("Missing or invalid user login")
	}

	user, err := client.User(login)
	if err != nil {
		return err
	}

	tmpl, err := template.New("_").Parse(c.String("format") + "\n")
	if err != nil {
		return err
	}
	return tmpl.Execute(os.Stdout, user)
}

// template for user information
var tmplUserInfo = `User: {{ .Login }}
Email: {{ .Email }}`
