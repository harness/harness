package model

const (
	RemoteGithub           = "github.com"
	RemoteGitlab           = "gitlab.com"
	RemoteGithubEnterprise = "enterprise.github.com"
	RemoteBitbucket        = "bitbucket.org"
	RemoteStash            = "stash.atlassian.com"
	RemoteGogs             = "gogs"
)

type Remote struct {
	ID     int64  `meddler:"remote_id,pk"  json:"id"`
	Type   string `meddler:"remote_type"   json:"type"`
	Host   string `meddler:"remote_host"   json:"host"`
	URL    string `meddler:"remote_url"    json:"url"`
	API    string `meddler:"remote_api"    json:"api"`
	Client string `meddler:"remote_client" json:"client"`
	Secret string `meddler:"remote_secret" json:"secret"`
	Open   bool   `meddler:"remote_open"   json:"open"`
}
