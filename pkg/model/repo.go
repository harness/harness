package model

import (
	"fmt"
	"time"
)

const (
	ScmGit = "git"
	ScmHg  = "hg"
	ScmSvn = "svn"
)

const (
	HostBitbucket = "bitbucket.org"
	HostGoogle    = "code.google.com"
	HostCustom    = "custom"
)

const (
	DefaultBranchGit = "master"
	DefaultBranchHg  = "default"
	DefaultBranchSvn = "trunk"
)

const (
	githubRepoPattern           = "git://%s/%s/%s.git"
	githubRepoPatternPrivate    = "git@%s:%s/%s.git"
	bitbucketRepoPattern        = "https://bitbucket.org/%s/%s.git"
	bitbucketRepoPatternPrivate = "git@bitbucket.org:%s/%s.git"
)

type Repo struct {
	ID int64 `meddler:"id,pk" json:"id"`

	// the full, canonical name of the repository, for example:
	// github.com/bradrydzewski/go.stripe
	Slug string `meddler:"slug" json:"slug"`

	// the hosting service where the repository is stored,
	// such as github.com, bitbucket.org, etc
	Host string `meddler:"host" json:"host"`

	// the owner of the repository on the host system.
	// for example, the Github username.
	Owner string `meddler:"owner" json:"owner"`

	// URL-friendly version of a repository name on the
	// host system.
	Name string `meddler:"name" json:"name"`

	// A value of True indicates the repository is closed source,
	// while a value of False indicates the project is open source.
	Private bool `meddler:"private" json:"private"`

	// A value of True indicates the repository is disabled and
	// no builds should be executed
	Disabled bool `meddler:"disabled" json:"disabled"`

	// A value of True indicates that pull requests are disabled
	// for the repository and no builds will be executed
	DisabledPullRequest bool `meddler:"disabled_pr" json:"disabled_pr"`

	// indicates the type of repository, such as
	// Git, Mercurial, Subversion or Bazaar.
	SCM string `meddler:"scm" json:"scm"`

	// the repository URL, for example:
	// git://github.com/bradrydzewski/go.stripe.git
	URL string `meddler:"url" json:"url"`

	// username and password requires to authenticate
	// to the repository
	Username string `meddler:"username" json:"username"`
	Password string `meddler:"password" json:"password"`

	// RSA key pair that will injected into the virtual machine
	// .ssh/id_rsa and .ssh/id_rsa.pub files.
	PublicKey  string `meddler:"public_key"  json:"public_key"`
	PrivateKey string `meddler:"private_key" json:"public_key"`

	// Parameters stored external to the repository in YAML
	// format, injected into the Build YAML at runtime.
	Params map[string]string `meddler:"params,gob" json:"-"`

	// the amount of time, in seconds the build will execute
	// before exceeding its timelimit and being killed.
	Timeout int64 `meddler:"timeout" json:"timeout"`

	// Indicates the build should be executed in privileged
	// mode. This could, for example, be used to run Docker in Docker.
	Privileged bool `meddler:"privileged" json:"privileged"`

	// Foreign keys signify the User that created
	// the repository and team account linked to
	// the repository.
	UserID int64 `meddler:"user_id"  json:"user_id"`
	TeamID int64 `meddler:"team_id"  json:"team_id"`

	Created time.Time `meddler:"created,utctime" json:"created"`
	Updated time.Time `meddler:"updated,utctime" json:"updated"`
}

// Creates a new repository
func NewRepo(host, owner, name, scm, url string) (*Repo, error) {
	repo := Repo{}
	repo.URL = url
	repo.SCM = scm
	repo.Host = host
	repo.Owner = owner
	repo.Name = name
	repo.Slug = fmt.Sprintf("%s/%s/%s", host, owner, name)
	if err := repo.AddKeyPair(); err != nil {
		return nil, err
	}
	return &repo, nil
}

// Creates a new GitHub repository
func NewGitHubRepo(domain, owner, name string, private bool) (*Repo, error) {
	var url string
	switch private {
	case false:
		url = fmt.Sprintf(githubRepoPattern, domain, owner, name)
	case true:
		url = fmt.Sprintf(githubRepoPatternPrivate, domain, owner, name)
	}
	return NewRepo(domain, owner, name, ScmGit, url)
}

// Creates a new Bitbucket repository
func NewBitbucketRepo(owner, name string, private bool) (*Repo, error) {
	var url string
	switch private {
	case false:
		url = fmt.Sprintf(bitbucketRepoPattern, owner, name)
	case true:
		url = fmt.Sprintf(bitbucketRepoPatternPrivate, owner, name)
	}
	return NewRepo(HostBitbucket, owner, name, ScmGit, url)
}

func (r *Repo) DefaultBranch() string {
	switch r.SCM {
	case ScmGit:
		return DefaultBranchGit
	case ScmHg:
		return DefaultBranchHg
	case ScmSvn:
		return DefaultBranchSvn
	default:
		return DefaultBranchGit
	}
}

// AddKeyPair effectively replacing key pair for current repo.
// Optionally accepts `public key` and `private key` respectively
func (r *Repo) AddKeyPair(keys ...[]byte) error {
	switch len(keys) {
	case 0:
		key, err := generatePrivateKey()
		if err != nil {
			return err
		}
		r.PublicKey = marshalPublicKey(&key.PublicKey)
		r.PrivateKey = marshalPrivateKey(key)
	case 2:
		//XXX: Could use some key checking
		r.PublicKey = string(keys[0])
		r.PrivateKey = string(keys[1])
	default:
		return fmt.Errorf("Parameters not match. Should supply key pairs or none (keys will be auto-generated)")
	}
	return nil
}
