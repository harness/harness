package store

import (
	"io"

	common "github.com/drone/drone/pkg/types"
)

type Store interface {

	// User returns a user by user ID.
	User(id int64) (*common.User, error)

	// UserLogin returns a user by user login.
	UserLogin(string) (*common.User, error)

	// UserList returns a list of all registered users.
	UserList() ([]*common.User, error)

	// UserFeed retrieves a digest of recent builds
	// from the datastore accessible to the specified user.
	UserFeed(*common.User, int, int) ([]*common.RepoCommit, error)

	// UserCount returns a count of all registered users.
	UserCount() (int, error)

	// AddUser inserts a new user into the datastore.
	// If the user login already exists an error is returned.
	AddUser(*common.User) error

	// SetUser updates an existing user.
	SetUser(*common.User) error

	// DelUser removes the user from the datastore.
	DelUser(*common.User) error

	//

	// Token returns a token by ID.
	Token(int64) (*common.Token, error)

	// TokenLabel returns a token by label
	TokenLabel(*common.User, string) (*common.Token, error)

	// TokenList returns a list of all user tokens.
	TokenList(*common.User) ([]*common.Token, error)

	// AddToken inserts a new token into the datastore.
	// If the token label already exists for the user
	// an error is returned.
	AddToken(*common.Token) error

	// DelToken removes the DelToken from the datastore.
	DelToken(*common.Token) error

	//

	// Starred returns true if the user starred
	// the given repository.
	Starred(*common.User, *common.Repo) (bool, error)

	// AddStar stars a repository.
	AddStar(*common.User, *common.Repo) error

	// DelStar unstars a repository.
	DelStar(*common.User, *common.Repo) error

	//

	// Repo retrieves a specific repo from the
	// datastore for the given ID.
	Repo(id int64) (*common.Repo, error)

	// RepoName retrieves a repo from the datastore
	// for the specified name.
	RepoName(owner, name string) (*common.Repo, error)

	// RepoList retrieves a list of all repos from
	// the datastore accessible by the given user ID.
	RepoList(*common.User) ([]*common.Repo, error)

	// AddRepo inserts a repo in the datastore.
	AddRepo(*common.Repo) error

	// SetRepo updates a repo in the datastore.
	SetRepo(*common.Repo) error

	// DelRepo removes the repo from the datastore.
	DelRepo(*common.Repo) error

	//

	// Commit gets a commit by ID
	Commit(int64) (*common.Commit, error)

	// CommitSeq gets the specified commit sequence for the
	// named repository and commit number
	CommitSeq(*common.Repo, int) (*common.Commit, error)

	// CommitLast gets the last executed commit for the
	// named repository and branch
	CommitLast(*common.Repo, string) (*common.Commit, error)

	// CommitList gets a list of recent commits for the
	// named repository.
	CommitList(*common.Repo, int, int) ([]*common.Commit, error)

	// AddCommit inserts a new commit in the datastore.
	AddCommit(*common.Commit) error

	// SetCommit updates an existing commit and commit tasks.
	SetCommit(*common.Commit) error

	// KillCommits updates all pending or started commits
	// in the datastore settings the status to killed.
	KillCommits() error

	//

	// Build returns a build by ID.
	Build(int64) (*common.Build, error)

	// BuildSeq returns a build by sequence number.
	BuildSeq(*common.Commit, int) (*common.Build, error)

	// BuildList returns a list of all commit builds
	BuildList(*common.Commit) ([]*common.Build, error)

	// SetBuild updates an existing build.
	SetBuild(*common.Build) error

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
	Agent(*common.Commit) (string, error)

	// SetAgent updates an agent in the datastore.
	SetAgent(*common.Commit, string) error
}
