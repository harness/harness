package repo

var DefaultBranch = "master"

// default build timeout, in seconds
var DefaultTimeout int64 = 7200

const (
	HostGitlab           = "gitlab.com"
	HostGithub           = "github.com"
	HostGithubEnterprise = "enterprise.github.com"
	HostBitbucket        = "bitbucket.org"
)

type Repo struct {
	ID     int64  `meddler:"repo_id,pk"        json:"-"`
	UserID int64  `meddler:"user_id"           json:"-"`
	Remote string `meddler:"repo_remote"       json:"remote"`
	Owner  string `meddler:"repo_owner"        json:"owner"`
	Name   string `meddler:"repo_name"         json:"name"`

	FullName string `meddler:"-" json:"full_name"`
	Clone    string `meddler:"-" json:"clone_url"`
	Git      string `meddler:"-" json:"git_url"`
	SSH      string `meddler:"-" json:"ssh_url"`

	URL         string `meddler:"repo_url"          json:"url"`
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
}

func NewGithub(owner, name string) (*Repo, error) {
	return New(HostGithub, owner, name)
}

func NewGithubEnterprise(owner, name string) (*Repo, error) {
	return New(HostGithubEnterprise, owner, name)
}

func NewGitlab(owner, name string) (*Repo, error) {
	return New(HostGitlab, owner, name)
}

func NewBitbucket(owner, name string) (*Repo, error) {
	return New(HostBitbucket, owner, name)
}

func New(remote, owner, name string) (*Repo, error) {
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
