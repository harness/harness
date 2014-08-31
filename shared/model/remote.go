package model

const (
	RemoteGithub           = "github.com"
	RemoteGitlab           = "gitlab.com"
	RemoteGithubEnterprise = "enterprise.github.com"
	RemoteBitbucket        = "bitbucket.org"
	RemoteStash            = "stash.atlassian.com"
)

type Remote struct {
	Id     int64  `gorm:"primary_key:yes" json:"id"`
	Type   string `json:"type"`
	Host   string `json:"host"`
	Url    string `json:"url"`
	Api    string `json:"api"`
	Client string `json:"client"`
	Secret string `json:"secret"`
	Open   bool   `json:"open"`
}
