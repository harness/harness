package store

import (
	"io"

	"github.com/drone/drone/pkg/types"

	log "github.com/drone/drone/Godeps/_workspace/src/github.com/Sirupsen/logrus"
)

var drivers = make(map[string]DriverFunc)

// Register makes a datastore driver available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, driver DriverFunc) {
	if driver == nil {
		panic("datastore: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("datastore: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

// DriverFunc returns a new connection to the datastore.
// The name is a string in a driver-specific format.
type DriverFunc func(driver, datasource string) (Store, error)

// New creates a new database connection specified by its database driver
// name and a driver-specific data source name, usually consisting of at
// least a database name and connection information.
func New(driver, datasource string) (Store, error) {
	fn, ok := drivers[driver]
	if !ok {
		log.Fatalf("datastore: unknown driver %q", driver)
	}
	log.Infof("datastore: loading driver %s", driver)
	log.Infof("datastore: loading config %s", datasource)
	return fn(driver, datasource)
}

type Store interface {

	// User returns a user by user ID.
	User(id int64) (*types.User, error)

	// UserLogin returns a user by user login.
	UserLogin(string) (*types.User, error)

	// UserList returns a list of all registered users.
	UserList() ([]*types.User, error)

	// UserFeed retrieves a digest of recent builds
	// from the datastore accessible to the specified user.
	UserFeed(*types.User, int, int) ([]*types.RepoCommit, error)

	// UserCount returns a count of all registered users.
	UserCount() (int, error)

	// AddUser inserts a new user into the datastore.
	// If the user login already exists an error is returned.
	AddUser(*types.User) error

	// SetUser updates an existing user.
	SetUser(*types.User) error

	// DelUser removes the user from the datastore.
	DelUser(*types.User) error

	//

	// Token returns a token by ID.
	Token(int64) (*types.Token, error)

	// TokenLabel returns a token by label
	TokenLabel(*types.User, string) (*types.Token, error)

	// TokenList returns a list of all user tokens.
	TokenList(*types.User) ([]*types.Token, error)

	// AddToken inserts a new token into the datastore.
	// If the token label already exists for the user
	// an error is returned.
	AddToken(*types.Token) error

	// DelToken removes the DelToken from the datastore.
	DelToken(*types.Token) error

	//

	// Starred returns true if the user starred
	// the given repository.
	Starred(*types.User, *types.Repo) (bool, error)

	// AddStar stars a repository.
	AddStar(*types.User, *types.Repo) error

	// DelStar unstars a repository.
	DelStar(*types.User, *types.Repo) error

	//

	// Repo retrieves a specific repo from the
	// datastore for the given ID.
	Repo(id int64) (*types.Repo, error)

	// RepoName retrieves a repo from the datastore
	// for the specified name.
	RepoName(owner, name string) (*types.Repo, error)

	// RepoList retrieves a list of all repos from
	// the datastore accessible by the given user ID.
	RepoList(*types.User) ([]*types.Repo, error)

	// AddRepo inserts a repo in the datastore.
	AddRepo(*types.Repo) error

	// SetRepo updates a repo in the datastore.
	SetRepo(*types.Repo) error

	// DelRepo removes the repo from the datastore.
	DelRepo(*types.Repo) error

	//

	// Build gets a build by ID
	Build(int64) (*types.Build, error)

	// BuildNumber gets the specified build number for the
	// named repository and build number
	BuildNumber(*types.Repo, int) (*types.Build, error)

	// BuildPullRequestNumber gets the specified build number for the
	// named repository and build number
	BuildPullRequestNumber(*types.Repo, int) (*types.Build, error)

	// BuildSha gets the specified build number for the
	// named repository and sha
	BuildSha(*types.Repo, string, string) (*types.Build, error)

	// BuildLast gets the last executed build for the
	// named repository and branch
	BuildLast(*types.Repo, string) (*types.Build, error)

	// BuildList gets a list of recent builds for the
	// named repository.
	BuildList(*types.Repo, int, int) ([]*types.Build, error)

	// AddBuild inserts a new build in the datastore.
	AddBuild(*types.Build) error

	// SetBuild updates an existing build and build jobs.
	SetBuild(*types.Build) error

	// KillBuilds updates all pending or started builds
	// in the datastore settings the status to killed.
	KillBuilds() error

	//

	// Build returns a build by ID.
	Job(int64) (*types.Job, error)

	// JobNumber returns a jobs by sequence number.
	JobNumber(*types.Build, int) (*types.Job, error)

	// JobList returns a list of all build jobs
	JobList(*types.Build) ([]*types.Job, error)

	// SetJob updates an existing job.
	SetJob(*types.Job) error

	//

	// Get retrieves an object from the blobstore.
	GetBlob(path string) ([]byte, error)

	// GetBlobReader retrieves an object from the blobstore.
	// It is the caller's responsibility to call Close on
	// the ReadCloser when finished reading.
	GetBlobReader(path string) (io.ReadCloser, error)

	// Set inserts an object into the blobstore.
	SetBlob(path string, data []byte) error

	// SetBlobReader inserts an object into the blobstore by
	// consuming data from r until EOF.
	SetBlobReader(path string, r io.Reader) error

	// Del removes an object from the blobstore.
	DelBlob(path string) error

	//

	// Agent returns an agent by ID.
	Agent(*types.Build) (string, error)

	// SetAgent updates an agent in the datastore.
	SetAgent(*types.Build, string) error
}
