package github

import (
	"os"

	"github.com/drone/drone/plugin/remote"
)

func init() {
	init_github()
	init_github_enterprise()
}

// registers the GitHub (github.com) plugin
func init_github() {
	var cli = os.Getenv("GITHUB_CLIENT")
	var sec = os.Getenv("GITHUB_SECRET")
	if len(cli) == 0 ||
		len(sec) == 0 {
		return
	}
	var github = NewDefault(cli, sec)
	remote.Register(github)
}

// registers the GitHub Enterprise plugin
func init_github_enterprise() {
	var url = os.Getenv("GITHUB_ENTERPRISE_URL")
	var api = os.Getenv("GITHUB_ENTERPRISE_API")
	var cli = os.Getenv("GITHUB_ENTERPRISE_CLIENT")
	var sec = os.Getenv("GITHUB_ENTERPRISE_SECRET")

	if len(url) == 0 ||
		len(api) == 0 ||
		len(cli) == 0 ||
		len(sec) == 0 {
		return
	}
	var github = New(url, api, cli, sec)
	remote.Register(github)
}
