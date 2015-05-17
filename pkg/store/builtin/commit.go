package builtin

import (
	"database/sql"
	"time"

	common "github.com/drone/drone/pkg/types"
	"github.com/russross/meddler"
)

type Commitstore struct {
	*sql.DB
}

func NewCommitstore(db *sql.DB) *Commitstore {
	return &Commitstore{db}
}

// Commit gets a commit by ID
func (db *Commitstore) Commit(id int64) (*common.Commit, error) {
	var commit = new(common.Commit)
	var err = meddler.Load(db, commitTable, commit, id)
	return commit, err
}

// CommitSeq gets the specified commit sequence for the
// named repository and commit number
func (db *Commitstore) CommitSeq(repo *common.Repo, seq int) (*common.Commit, error) {
	var commit = new(common.Commit)
	var err = meddler.QueryRow(db, commit, rebind(commitNumberQuery), repo.ID, seq)
	return commit, err
}

// CommitLast gets the last executed commit for the
// named repository.
func (db *Commitstore) CommitLast(repo *common.Repo, branch string) (*common.Commit, error) {
	var commit = new(common.Commit)
	var err = meddler.QueryRow(db, commit, rebind(commitLastQuery), repo.ID, branch)
	return commit, err
}

// CommitList gets a list of recent commits for the
// named repository.
func (db *Commitstore) CommitList(repo *common.Repo, limit, offset int) ([]*common.Commit, error) {
	var commits []*common.Commit
	var err = meddler.QueryAll(db, &commits, rebind(commitListQuery), repo.ID, limit, offset)
	return commits, err
}

// AddCommit inserts a new commit in the datastore.
func (db *Commitstore) AddCommit(commit *common.Commit) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// extract the next commit number from the database
	row := tx.QueryRow(rebind(commitNumberLast), commit.RepoID)
	if row != nil {
		row.Scan(&commit.Sequence)
	}

	commit.Sequence = commit.Sequence + 1 // increment
	commit.Created = time.Now().UTC().Unix()
	commit.Updated = time.Now().UTC().Unix()
	err = meddler.Insert(tx, commitTable, commit)
	if err != nil {
		return err
	}

	for _, build := range commit.Builds {
		build.CommitID = commit.ID
		build.Created = commit.Created
		build.Updated = commit.Updated
		err := meddler.Insert(tx, buildTable, build)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SetCommit updates an existing commit and commit tasks.
func (db *Commitstore) SetCommit(commit *common.Commit) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	commit.Updated = time.Now().UTC().Unix()
	err = meddler.Update(tx, commitTable, commit)
	if err != nil {
		return err
	}

	for _, build := range commit.Builds {
		build.Updated = commit.Updated
		err := meddler.Update(tx, buildTable, build)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// KillCommits updates all pending or started commits
// in the datastore settings the status to killed.
func (db *Commitstore) KillCommits() error {
	var _, err1 = db.Exec(rebind(buildKillStmt))
	if err1 != nil {
		return err1
	}
	var _, err2 = db.Exec(rebind(commitKillStmt))
	return err2
}

// Commit table name in database.
const commitTable = "commits"

// SQL query to retrieve the latest commits across all branches.
const commitListQuery = `
SELECT *
FROM commits
WHERE repo_id = ?
ORDER BY commit_seq DESC
LIMIT ? OFFSET ?
`

// SQL query to retrieve a commit by number.
const commitNumberQuery = `
SELECT *
FROM commits
WHERE repo_id    = ?
  AND commit_seq = ?
LIMIT 1
`

// SQL query to retrieve the most recent commit.
// TODO exclude pull requests
const commitLastQuery = `
SELECT *
FROM commits
WHERE repo_id       = ?
  AND commit_branch = ?
ORDER BY commit_seq DESC
LIMIT 1
`

// SQL statement to cancel all running commits.
const commitKillStmt = `
UPDATE commits SET commit_state = 'killed'
WHERE commit_state IN ('pending', 'running');
`

// SQL statement to cancel all running commits.
const buildKillStmt = `
UPDATE builds SET build_state = 'killed'
WHERE build_state IN ('pending', 'running');
`

// SQL statement to retrieve the commit number for
// a commit
const commitNumberLast = `
SELECT MAX(commit_seq)
FROM commits 
WHERE repo_id = ?
`
