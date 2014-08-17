package model

import (
	"gopkg.in/yaml.v1"
)

var (
	DefaultBranch = "master"

	// default build timeout, in seconds
	DefaultTimeout int64 = 7200
)

// RepoParams represents a set of private key value parameters
// for each Repository.
type RepoParams map[string]string

type Repo struct {
	Id     int64  `gorm:"primary_key:yes"        json:"-"`
	UserId int64  `json:"-"`
	Remote string `json:"remote"`
	Host   string `json:"host"`
	Owner  string `json:"owner"`
	Name   string `json:"name"`

	Url      string `json:"url"`
	CloneUrl string `json:"clone_url"`
	GitUrl   string `json:"git_url"`
	SshUrl   string `json:"ssh_url"`

	Active      bool   `json:"active"`
	Private     bool   `json:"private"`
	Privileged  bool   `json:"privileged"`
	PostCommit  bool   `json:"post_commits"`
	PullRequest bool   `json:"pull_requests"`
	PublicKey   string `json:"-"`
	PrivateKey  string `json:"-"`
	Params      string `json:"-"`
	Timeout     int64  `json:"timeout"`
	Created     int64  `json:"created_at"`
	Updated     int64  `json:"updated_at"`
}

func NewRepo(remote, owner, name string) (*Repo, error) {
	repo := Repo{}
	repo.Remote = remote
	repo.Owner = owner
	repo.Name = name
	repo.Active = false
	repo.PostCommit = true
	repo.PullRequest = true
	repo.Timeout = DefaultTimeout
	return &repo, nil
}

func (r *Repo) ParamMap() (map[string]string, error) {
	out := map[string]string{}
	err := yaml.Unmarshal([]byte(r.Params), out)
	return out, err
}
