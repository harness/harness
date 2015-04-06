package gogs

import (
	"github.com/drone/config"
	"github.com/drone/drone/plugin/remote"
)

var (
	gogsUrl    = config.String("gogs-url", "")
	gogsSecret = config.String("gogs-secret", "")
	gogsOpen   = config.Bool("gogs-open", false)
)

// Registers the Gogs plugin using the default
// settings from the config file or environment
// variables.
func Register() {
	if len(*gogsUrl) == 0 {
		return
	}
	remote.Register(
		New(*gogsUrl, *gogsSecret, *gogsOpen),
	)
}
