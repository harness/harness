package gitlab

import (
	"github.com/drone/config"
	"github.com/drone/drone/plugin/remote"
)

var (
	gitlabURL        = config.String("gitlab-url", "")
	gitlabSkipVerify = config.Bool("gitlab-skip-verify", false)
	gitlabOpen       = config.Bool("gitlab-open", false)

	gitlabClient = config.String("gitlab-client", "")
	gitlabSecret = config.String("gitlab-secret", "")
)

// Registers the Gitlab plugin using the default
// settings from the config file or environment
// variables.
func Register() {
	if len(*gitlabURL) == 0 {
		return
	}
	remote.Register(
		New(
			*gitlabURL,
			*gitlabSkipVerify,
			*gitlabOpen,
			*gitlabClient,
			*gitlabSecret,
		),
	)
}
