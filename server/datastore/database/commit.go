package database

import (
	"fmt"
	"time"

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
func (db *Commitstore) GetCommitList(repo *model.Repo, limit, offset int) ([]*model.Commit, error) {
	var commits []*model.Commit
	var err = meddler.QueryAll(db, &commits, rebind(commitListQuery), repo.ID, limit, offset)
	return commits, err
}

// GetCommitListUser retrieves a list of latest commits
// from the datastore accessible to the specified user.
func (db *Commitstore) GetCommitListUser(user *model.User) ([]*model.CommitRepo, error) {
	var commits []*model.CommitRepo
	var err = meddler.QueryAll(db, &commits, rebind(commitListUserQuery), user.ID)
	return commits, err
}

// GetCommitListActivity retrieves an ungrouped list of latest commits
// from the datastore accessible to the specified user.
func (db *Commitstore) GetCommitListActivity(user *model.User, limit, offset int) ([]*model.CommitRepo, error) {
	var commits []*model.CommitRepo
	var err = meddler.QueryAll(db, &commits, rebind(commitListActivityQuery), user.ID, limit, offset)
	return commits, err
}

// GetCommitPrior retrieves the latest commit
// from the datastore for the specified repository and branch.
func (db *Commitstore) GetCommitPrior(oldCommit *model.Commit) (*model.Commit, error) {
	var commit = new(model.Commit)
	var err = meddler.QueryRow(db, commit, rebind(commitPriorQuery), oldCommit.RepoID, oldCommit.Branch, oldCommit.ID)
	return commit, err
}

// PostCommit saves a commit in the datastore.
func (db *Commitstore) PostCommit(commit *model.Commit) error {
	if commit.Created == 0 {
		commit.Created = time.Now().UTC().Unix()
	}
	commit.Updated = time.Now().UTC().Unix()
	return meddler.Save(db, commitTable, commit)
}

// PutCommit saves a commit in the datastore.
func (db *Commitstore) PutCommit(commit *model.Commit) error {
	if commit.Created == 0 {
		commit.Created = time.Now().UTC().Unix()
	}
	commit.Updated = time.Now().UTC().Unix()
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

// GetBuildNumber retrieves the build number for a commit.
func (db *Commitstore) GetBuildNumber(commit *model.Commit) (int64, error) {
	row := db.QueryRow(rebind(commitGetBuildNumberStmt), commit.ID, commit.RepoID)
	if row == nil {
		return 0, fmt.Errorf("Unable to get build number for commit %d", commit.ID)
	}
	var bn int64
	err := row.Scan(&bn)
	if err != nil {
		return 0, err
	}
	return bn, nil
}

// Commit table name in database.
const commitTable = "commits"

// SQL statement to delete a Commit by ID.
const commitDeleteStmt = `
DELETE FROM commits
WHERE commit_id = ?
`

// SQL query to retrieve the latest Commits accessible
// to a specific user account
const commitListUserQuery = `
SELECT r.repo_remote, r.repo_host, r.repo_owner, r.repo_name, c.*
FROM
 commits c
,repos r
WHERE c.repo_id = r.repo_id
  AND c.commit_id IN (
	SELECT max(c.commit_id)
	FROM
	 commits c
	,repos r
	,perms p
	WHERE c.repo_id = r.repo_id
	  AND r.repo_id = p.repo_id
	  AND p.user_id = ?
	GROUP BY r.repo_id
) ORDER BY c.commit_created DESC;
`

// SQL query to retrieve the ungrouped, latest Commits
// accessible to a specific user account
const commitListActivityQuery = `
SELECT r.repo_remote, r.repo_host, r.repo_owner, r.repo_name, c.*
FROM
 commits c
,repos r
,perms p
WHERE c.repo_id = r.repo_id
  AND r.repo_id = p.repo_id
  AND p.user_id = ?
ORDER BY c.commit_created DESC
LIMIT ? OFFSET ?
`

// SQL query to retrieve the latest Commits across all branches.
const commitListQuery = `
SELECT *
FROM commits
WHERE repo_id = ?
ORDER BY commit_id DESC
LIMIT ? OFFSET ?
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
  AND commit_pr     = ''
ORDER BY commit_id DESC
LIMIT 1
`

// SQL query to retrieve the prior Commit (by commit_created) in the same branch and repo as the specified Commit.
const commitPriorQuery = `
SELECT *
FROM commits
WHERE repo_id       = ?
  AND commit_branch = ?
  AND commit_id     < ?
ORDER BY commit_id DESC
LIMIT 1
`

// SQL statement to cancel all running Commits.
const commitKillStmt = `
UPDATE commits SET commit_status = 'Killed'
WHERE commit_status IN ('Started', 'Pending');
`

// SQL statement to retrieve the build number for
// a commit
const commitGetBuildNumberStmt = `
SELECT COUNT(1)
FROM commits 
WHERE commit_id <= ? 
	AND repo_id = ?
`
