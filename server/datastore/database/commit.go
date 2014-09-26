package database

import (
	"github.com/drone/drone/shared/model"
	"github.com/russross/meddler"
)

type Commitstore struct {
	meddler.DB
}

func NewCommitstore(db meddler.DB) *Commitstore {
	return &Commitstore{db}
}

// GetCommit retrieves a commit from the
// datastore for the given ID.
func (db *Commitstore) GetCommit(id int64) (*model.Commit, error) {
	var commit = new(model.Commit)
	var err = meddler.Load(db, commitTable, commit, id)
	return commit, err
}

// GetCommitSha retrieves a commit from the
// datastore for the specified repo and sha
func (db *Commitstore) GetCommitSha(repo *model.Repo, branch, sha string) (*model.Commit, error) {
	var commit = new(model.Commit)
	var err = meddler.QueryRow(db, commit, rebind(commitShaQuery), repo.ID, branch, sha)
	return commit, err
}

// GetCommitLast retrieves the latest commit
// from the datastore for the specified repository
// and branch.
func (db *Commitstore) GetCommitLast(repo *model.Repo, branch string) (*model.Commit, error) {
	var commit = new(model.Commit)
	var err = meddler.QueryRow(db, commit, rebind(commitLastQuery), repo.ID, branch)
	return commit, err
}

// GetCommitList retrieves a list of latest commits
// from the datastore for the specified repository.
func (db *Commitstore) GetCommitList(repo *model.Repo) ([]*model.Commit, error) {
	var commits []*model.Commit
	var err = meddler.QueryAll(db, &commits, rebind(commitListQuery))
	return commits, err
}

// GetCommitListUser retrieves a list of latest commits
// from the datastore accessible to the specified user.
func (db *Commitstore) GetCommitListUser(user *model.User) ([]*model.Commit, error) {
	return nil, nil
}

// PostCommit saves a commit in the datastore.
func (db *Commitstore) PostCommit(commit *model.Commit) error {
	return meddler.Save(db, commitTable, commit)
}

// PutCommit saves a commit in the datastore.
func (db *Commitstore) PutCommit(commit *model.Commit) error {
	return meddler.Save(db, commitTable, commit)
}

// DelCommit removes the commit from the datastore.
func (db *Commitstore) DelCommit(commit *model.Commit) error {
	var _, err = db.Exec(rebind(commitDeleteStmt), commit.ID)
	return err
}

// KillCommits updates all pending or started commits
// in the datastore settings the status to killed.
func (db *Commitstore) KillCommits() error {
	var _, err = db.Exec(rebind(commitKillStmt))
	return err
}

// Commit table name in database.
const commitTable = "commits"

// SQL statement to delete a Commit by ID.
const commitDeleteStmt = `
DELETE FROM commits
WHERE commit_id = ?
`

// SQL query to retrieve the latest Commits across all branches.
const commitListQuery = `
SELECT *
FROM commits
WHERE repo_id = ? 
ORDER BY commit_id DESC
LIMIT 20
`

// SQL query to retrieve a Commit by branch and sha.
const commitShaQuery = `
SELECT *
FROM commits
WHERE repo_id       = ?
  AND commit_branch = ?
  AND commit_sha    = ?
LIMIT 1
`

// SQL query to retrieve the most recent Commit for a branch.
const commitLastQuery = `
SELECT *
FROM commits
WHERE repo_id       = ?
  AND commit_branch = ?
ORDER BY commit_id DESC
LIMIT 1
`

// SQL statement to cancel all running Commits.
const commitKillStmt = `
UPDATE commits SET
commit_status   = ?,
commit_started  = ?,
commit_finished = ?
WHERE commit_status IN ('Started', 'Pending');
`
