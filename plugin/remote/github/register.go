package github

import (
	"github.com/drone/config"
	"github.com/drone/drone/plugin/remote"
)

var (
	// GitHub cloud configuration details
	githubClient = config.String("github-client", "")
	githubSecret = config.String("github-secret", "")

	// GitHub Enterprise configuration details
	githubEnterpriseURL     = config.String("github-enterprise-url", "")
	githubEnterpriseAPI     = config.String("github-enterprise-api", "")
	githubEnterpriseClient  = config.String("github-enterprise-client", "")
	githubEnterpriseSecret  = config.String("github-enterprise-secret", "")
	githubEnterprisePrivate = config.Bool("github-enterprise-private-mode", true)
)

// Registers the GitHub plugins using the default
// settings from the config file or environment
// variables.
func Register() {
	registerGitHub()
	registerGitHubEnterprise()
}

// registers the GitHub (github.com) plugin
func registerGitHub() {
	if len(*githubClient) == 0 || len(*githubSecret) == 0 {
		return
	}
	remote.Register(
		NewDefault(*githubClient, *githubSecret),
	)
}

// registers the GitHub Enterprise plugin
func registerGitHubEnterprise() {
	if len(*githubEnterpriseURL) == 0 ||
		len(*githubEnterpriseAPI) == 0 ||
		len(*githubEnterpriseClient) == 0 ||
		len(*githubEnterpriseSecret) == 0 {
		return
	}
	remote.Register(
		New(
			*githubEnterpriseURL,
			*githubEnterpriseAPI,
			*githubEnterpriseClient,
			*githubEnterpriseSecret,
			*githubEnterprisePrivate,
		),
	)
}
