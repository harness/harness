package bitbucket

import (
	"github.com/drone/config"
	"github.com/drone/drone/plugin/remote"
)

var (
	// Bitbucket cloud configuration details
	bitbucketClient = config.String("bitbucket-client", "")
	bitbucketSecret = config.String("bitbucket-secret", "")
	bitbucketOpen   = config.Bool("bitbucket-open", false)
)

// Registers the Bitbucket plugin using the default
// settings from the config file or environment
// variables.
func Register() {
	if len(*bitbucketClient) == 0 || len(*bitbucketSecret) == 0 {
		return
	}
	remote.Register(
		NewDefault(*bitbucketClient, *bitbucketSecret, *bitbucketOpen),
	)
}
