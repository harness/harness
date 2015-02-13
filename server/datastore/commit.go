package datastore

import (
	"code.google.com/p/go.net/context"
	"github.com/drone/drone/shared/model"
)

type Commitstore interface {
	// GetCommit retrieves a commit from the
	// datastore for the given ID.
	GetCommit(id int64) (*model.Commit, error)

	// GetCommitSha retrieves a commit from the
	// datastore for the specified repo and sha
	GetCommitSha(repo *model.Repo, branch, sha string) (*model.Commit, error)

	// GetCommitLast retrieves the latest commit
	// from the datastore for the specified repository
	// and branch.
	GetCommitLast(repo *model.Repo, branch string) (*model.Commit, error)

	// GetCommitList retrieves a list of latest commits
	// from the datastore for the specified repository.
	GetCommitList(repo *model.Repo, limit, offset int) ([]*model.Commit, error)

	// GetCommitListUser retrieves a list of latest commits
	// from the datastore accessible to the specified user.
	GetCommitListUser(user *model.User) ([]*model.CommitRepo, error)

	// GetCommitListActivity retrieves an ungrouped list of latest commits
	// from the datastore accessible to the specified user.
	GetCommitListActivity(user *model.User, limit, offset int) ([]*model.CommitRepo, error)

	// GetCommitPrior retrieves the latest commit
	// from the datastore for the specified repository and branch.
	GetCommitPrior(commit *model.Commit) (*model.Commit, error)

	// PostCommit saves a commit in the datastore.
	PostCommit(commit *model.Commit) error

	// PutCommit saves a commit in the datastore.
	PutCommit(commit *model.Commit) error

	// DelCommit removes the commit from the datastore.
	DelCommit(commit *model.Commit) error

	// KillCommits updates all pending or started commits
	// in the datastore settings the status to killed.
	KillCommits() error

	// GetCommitBuildNumber retrieves the monotonically increaing build number
	// from the commit's repo
	GetBuildNumber(commit *model.Commit) (int64, error)
}

// GetCommit retrieves a commit from the
// datastore for the given ID.
func GetCommit(c context.Context, id int64) (*model.Commit, error) {
	return FromContext(c).GetCommit(id)
}

// GetCommitSha retrieves a commit from the
// datastore for the specified repo and sha
func GetCommitSha(c context.Context, repo *model.Repo, branch, sha string) (*model.Commit, error) {
	return FromContext(c).GetCommitSha(repo, branch, sha)
}

// GetCommitLast retrieves the latest commit
// from the datastore for the specified repository
// and branch.
func GetCommitLast(c context.Context, repo *model.Repo, branch string) (*model.Commit, error) {
	return FromContext(c).GetCommitLast(repo, branch)
}

// GetCommitList retrieves a list of latest commits
// from the datastore for the specified repository.
func GetCommitList(c context.Context, repo *model.Repo, limit, offset int) ([]*model.Commit, error) {
	return FromContext(c).GetCommitList(repo, limit, offset)
}

// GetCommitListUser retrieves a list of latest commits
// from the datastore accessible to the specified user.
func GetCommitListUser(c context.Context, user *model.User) ([]*model.CommitRepo, error) {
	return FromContext(c).GetCommitListUser(user)
}

// GetCommitListActivity retrieves an ungrouped list of latest commits
// from the datastore accessible to the specified user.
func GetCommitListActivity(c context.Context, user *model.User, limit, offset int) ([]*model.CommitRepo, error) {
	return FromContext(c).GetCommitListActivity(user, limit, offset)
}

// GetCommitPrior retrieves the latest commit
// from the datastore for the specified repository and branch.
func GetCommitPrior(c context.Context, commit *model.Commit) (*model.Commit, error) {
	return FromContext(c).GetCommitPrior(commit)
}

// PostCommit saves a commit in the datastore.
func PostCommit(c context.Context, commit *model.Commit) error {
	return FromContext(c).PostCommit(commit)
}

// PutCommit saves a commit in the datastore.
func PutCommit(c context.Context, commit *model.Commit) error {
	return FromContext(c).PutCommit(commit)
}

// DelCommit removes the commit from the datastore.
func DelCommit(c context.Context, commit *model.Commit) error {
	return FromContext(c).DelCommit(commit)
}

// KillCommits updates all pending or started commits
// in the datastore settings the status to killed.
func KillCommits(c context.Context) error {
	return FromContext(c).KillCommits()
}

// GetBuildNumber retrieves the monotonically increaing build number
// from the commit's repo
func GetBuildNumber(c context.Context, commit *model.Commit) (int64, error) {
	return FromContext(c).GetBuildNumber(commit)
}
