package hooks

import (
	"github.com/harness/gitness/client"
	"gopkg.in/alecthomas/kingpin.v2"
)

func Register(app *kingpin.Application, client client.Client) {
	cmd := app.Command("hooks", "manage git server hooks")
	registerUpdate(cmd, client)
}
