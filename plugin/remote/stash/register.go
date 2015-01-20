package stash

import (
	"github.com/drone/config"
	"github.com/drone/drone/plugin/remote"
)

var (
	// Stash configuration details
	stashURL        = config.String("stash-url", "")
	stashAPI        = config.String("stash-api", "")
	stashSecret     = config.String("stash-secret", "")
	stashPrivateKey = config.String("stash-private-key", "")
	stashHook       = config.String("stash-hook", "")
	stashOpen       = config.Bool("stash-open", false)
)

// Registers the Stash plugin using the default
// settings from the config file or environment
// variables
func Register() {
	if len(*stashURL) == 0 ||
		len(*stashAPI) == 0 ||
		len(*stashSecret) == 0 ||
		len(*stashPrivateKey) == 0 ||
		len(*stashHook) == 0 {
		return
	}
	remote.Register(
		New(
			*stashURL,
			*stashAPI,
			*stashSecret,
			*stashPrivateKey,
			*stashHook,
			*stashOpen,
		),
	)
}
