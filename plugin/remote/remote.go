package remote

import (
	"net/http"

	"github.com/drone/drone/shared/model"
)

// Defines a model for integrating (or pluggin in) remote version
// control systems, such as GitHub and Bitbucket.
type Plugin func(*model.Remote) Remote

var plugins = map[string]Plugin{}

// Register registers a new plugin.
func Register(name string, plugin Plugin) {
	plugins[name] = plugin
}

// Lookup retrieves the plugin for the remote.
func Lookup(name string) (Plugin, bool) {
	plugin, ok := plugins[name]
	return plugin, ok
}

type Remote interface {
	// GetName returns the name of this remote system.
	GetName() string

	// GetHost returns the URL hostname of this remote system.
	GetHost() (host string)

	// GetHook parses the post-commit hook from the Request body
	// and returns the required data in a standard format.
	GetHook(*http.Request, *model.User) (*Hook, error)

	// GetLogin handles authentication to third party, remote services
	// and returns the required user data in a standard format.
	GetLogin(http.ResponseWriter, *http.Request) (*Login, error)

	// NewClient returns a new Bitbucket remote client.
	GetClient(access, secret string) Client

	// Match returns true if the hostname matches the
	// hostname of this remote client.
	IsMatch(hostname string) bool
}

type Client interface {
	// GetUser fetches the user by ID (login name).
	GetUser(login string) (*User, error)

	// GetRepos fetches all repositories that the specified
	// user has access to in the remote system.
	GetRepos(owner string) ([]*Repo, error)

	// GetScript fetches the build script (.drone.yml) from the remote
	// repository and returns in string format.
	GetScript(*Hook) (string, error)

	// SetStatus
	SetStatus(owner, repo, sha, status string) error

	// SetActive
	SetActive(owner, repo, hook, key string) error
}

// Hook represents a subset of commit meta-data provided
// by post-commit and pull request hooks.
type Hook struct {
	Owner       string
	Repo        string
	Sha         string
	Branch      string
	PullRequest string
	Author      string
	Gravatar    string
	Timestamp   string
	Message     string
}

// Login represents a standard subset of user meta-data
// provided by OAuth login services.
type Login struct {
	ID     int64
	Login  string
	Access string
	Secret string
	Name   string
	Email  string
}

// User represents a standard subset of user meta-data
// returned by REST API user endpoints (ie github user api).
type User struct {
	ID       int64
	Login    string
	Name     string
	Gravatar string
}

// Repo represents a standard subset of repository meta-data
// returned by REST API endpoints (ie github repo api).
type Repo struct {
	ID      int64
	Host    string
	Owner   string
	Name    string
	Kind    string
	Clone   string
	Git     string
	SSH     string
	URL     string
	Private bool
	Pull    bool
	Push    bool
	Admin   bool
}
