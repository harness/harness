package model

import (
	"gopkg.in/yaml.v1"
)

const (
	Git       = "git"
	Mercurial = "mercurial"
)

var (
	// default build timeout, in seconds
	DefaultTimeout int64 = 7200
)

// RepoParams represents a set of private key value parameters
// for each Repository.
type RepoParams map[string]string

type Repo struct {
	ID     int64  `meddler:"repo_id,pk"        json:"-"`
	UserID int64  `meddler:"user_id"           json:"-"`
	Token  string `meddler:"repo_token"        json:"-"`
	Remote string `meddler:"repo_remote"       json:"remote"`
	Host   string `meddler:"repo_host"         json:"host"`
	Owner  string `meddler:"repo_owner"        json:"owner"`
	Name   string `meddler:"repo_name"         json:"name"`
	Scm    string `meddler:"repo_scm"          json:"scm"`

	URL      string `meddler:"repo_url"       json:"url"`
	CloneURL string `meddler:"repo_clone_url" json:"clone_url"`
	GitURL   string `meddler:"repo_git_url"   json:"git_url"`
	SSHURL   string `meddler:"repo_ssh_url"   json:"ssh_url"`

	Active      bool   `meddler:"repo_active"       json:"active"`
	Private     bool   `meddler:"repo_private"      json:"private"`
	Privileged  bool   `meddler:"repo_privileged"   json:"privileged"`
	PostCommit  bool   `meddler:"repo_post_commit"  json:"post_commits"`
	PullRequest bool   `meddler:"repo_pull_request" json:"pull_requests"`
	PublicKey   string `meddler:"repo_public_key"   json:"-"`
	PrivateKey  string `meddler:"repo_private_key"  json:"-"`
	Params      string `meddler:"repo_params"       json:"-"`
	Timeout     int64  `meddler:"repo_timeout"      json:"timeout"`
	Created     int64  `meddler:"repo_created"      json:"created_at"`
	Updated     int64  `meddler:"repo_updated"      json:"updated_at"`

	// Role defines the user's role relative to this repository.
	// Note that this data is stored separately in the datastore,
	// and must be joined to populate.
	Role *Perm `meddler:"-" json:"role,omitempty"`
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

func (r *Repo) DefaultBranch() string {
	var branch = "master"
	if r.Scm == Mercurial {
		branch = "default"
	}
	return branch
}
