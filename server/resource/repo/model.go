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
	Host   string `meddler:"repo_host"         json:"host"`
	Owner  string `meddler:"repo_owner"        json:"owner"`
	Name   string `meddler:"repo_name"         json:"name"`

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
