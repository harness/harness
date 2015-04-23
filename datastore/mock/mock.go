package mock

import (
	"io"

	"github.com/drone/drone/common"
	"github.com/drone/drone/datastore"
)

type Datastore struct {
	Tokens map[string]*common.Token
}

func (db Datastore) User(string) (*common.User, error) {
	return &common.User{}, nil
}
func (db Datastore) UserCount() (int, error) {
	return 0, nil
}

// UserList returns a list of all registered users.
func (db Datastore) UserList() ([]*common.User, error) {
	return []*common.User{}, nil
}

// SetUser inserts or updates a user.
func (db Datastore) SetUser(*common.User) error {
	return nil
}

// SetUserNotExists inserts a new user into the datastore.
// If the user login already exists ErrConflict is returned.
func (db Datastore) SetUserNotExists(*common.User) error {
	return nil
}

// Del deletes the user.
func (db Datastore) DelUser(*common.User) error {
	return nil
}

// Token returns the token for the given user and label.
func (db Datastore) Token(user, label string) (*common.Token, error) {
	key := string(user + "/" + label)

	if val := db.Tokens[key]; val != nil {
		return val, nil
	} else {
		return nil, datastore.ErrKeyNotFound
	}
}

// TokenList returns a list of all tokens for the given
// user login.
func (db Datastore) TokenList(string) ([]*common.Token, error) {
	return []*common.Token{}, nil
}

// SetToken inserts a new user token in the datastore.
func (db Datastore) SetToken(*common.Token) error {
	return nil
}

// DelToken deletes the token.
func (db Datastore) DelToken(*common.Token) error {
	return nil
}

// Subscribed returns true if the user is subscribed
// to the named repository.
func (db Datastore) Subscribed(string, string) (bool, error) {
	return true, nil
}

// SetSubscriber inserts a subscriber for the named
// repository.
func (db Datastore) SetSubscriber(string, string) error {
	return nil
}

// DelSubscriber removes the subscriber by login for the
// named repository.
func (db Datastore) DelSubscriber(string, string) error {
	return nil
}

// Repo returns the repository with the given name.
func (db Datastore) Repo(string) (*common.Repo, error) {
	return &common.Repo{}, nil
}

// RepoList returns a list of repositories for the
// given user account.
func (db Datastore) RepoList(string) ([]*common.Repo, error) {
	return []*common.Repo{}, nil
}

// RepoParams returns the private environment parameters
// for the given repository.
func (db Datastore) RepoParams(string) (map[string]string, error) {
	return make(map[string]string), nil
}

// RepoKeypair returns the private and public rsa keys
// for the given repository.
func (db Datastore) RepoKeypair(string) (*common.Keypair, error) {
	return &common.Keypair{}, nil
}

// SetRepo inserts or updates a repository.
func (db Datastore) SetRepo(*common.Repo) error {
	return nil
}

// SetRepo updates a repository. If the repository
// already exists ErrConflict is returned.
func (db Datastore) SetRepoNotExists(*common.User, *common.Repo) error {
	return nil
}

// SetRepoParams inserts or updates the private
// environment parameters for the named repository.
func (db Datastore) SetRepoParams(string, map[string]string) error {
	return nil
}

// SetRepoKeypair inserts or updates the private and
// public keypair for the named repository.
func (db Datastore) SetRepoKeypair(string, *common.Keypair) error {
	return nil
}

// DelRepo deletes the repository.
func (db Datastore) DelRepo(*common.Repo) error {
	return nil
}

// Build gets the specified build number for the
// named repository and build number
func (db Datastore) Build(string, int) (*common.Build, error) {
	return &common.Build{}, nil
}

// BuildList gets a list of recent builds for the
// named repository.
func (db Datastore) BuildList(string) ([]*common.Build, error) {
	return []*common.Build{}, nil
}

// BuildLast gets the last executed build for the
// named repository.
func (db Datastore) BuildLast(string) (*common.Build, error) {
	return &common.Build{}, nil
}

// SetBuild inserts or updates a build for the named
// repository. The build number is incremented and
// assigned to the provided build.
func (db Datastore) SetBuild(string, *common.Build) error {
	return nil
}

// Status returns the status for the given repository
// and build number.
func (db Datastore) Status(string, int, string) (*common.Status, error) {
	return &common.Status{}, nil
}

// StatusList returned a list of all build statues for
// the given repository and build number.
func (db Datastore) StatusList(string, int) ([]*common.Status, error) {
	return []*common.Status{}, nil
}

// SetStatus inserts a new build status for the
// named repository and build number. If the status already
// exists an error is returned.
func (db Datastore) SetStatus(string, int, *common.Status) error {
	return nil
}

// LogReader gets the task logs at index N for
// the named repository and build number.
func (db Datastore) LogReader(string, int, int) (io.Reader, error) {
	return nil, nil
}

// SetLogs inserts or updates a task logs for the
// named repository and build number.
func (db Datastore) SetLogs(string, int, int, []byte) error {
	return nil
}

// Experimental

// SetBuildState updates an existing build's start time,
// finish time, duration and state. No other fields are
// updated.
func (db Datastore) SetBuildState(string, *common.Build) error {
	return nil
}

// SetBuildStatus appends a new build status to an
// existing build record.
func (db Datastore) SetBuildStatus(string, int, *common.Status) error {
	return nil
}

// SetBuildTask updates an existing build task. The build
// and task must already exist. If the task does not exist
// an error is returned.
func (db Datastore) SetBuildTask(string, int, *common.Task) error {
	return nil
}
