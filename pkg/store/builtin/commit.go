package builtin

import (
	"database/sql"
	"time"

	"github.com/drone/drone/pkg/types"
)

type Commitstore struct {
	*sql.DB
}

func NewCommitstore(db *sql.DB) *Commitstore {
	return &Commitstore{db}
}

// Commit gets a commit by ID
func (db *Commitstore) Commit(id int64) (*types.Commit, error) {
	return getCommit(db, rebind(stmtCommitSelect), id)
}

// CommitSeq gets the specified commit sequence for the
// named repository and commit number
func (db *Commitstore) CommitSeq(repo *types.Repo, seq int) (*types.Commit, error) {
	return getCommit(db, rebind(stmtCommitSelectCommitSeq), repo.ID, seq)
}

// CommitLast gets the last executed commit for the
// named repository.
func (db *Commitstore) CommitLast(repo *types.Repo, branch string) (*types.Commit, error) {
	return getCommit(db, rebind(commitLastQuery), repo.ID, branch)
}

// CommitList gets a list of recent commits for the
// named repository.
func (db *Commitstore) CommitList(repo *types.Repo, limit, offset int) ([]*types.Commit, error) {
	return getCommits(db, rebind(commitListQuery), repo.ID, limit, offset)
}

// AddCommit inserts a new commit in the datastore.
func (db *Commitstore) AddCommit(commit *types.Commit) error {
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
	err = createCommit(tx, rebind(stmtCommitInsert), commit)
	if err != nil {
		return err
	}

	for _, build := range commit.Builds {
		build.CommitID = commit.ID
		build.Created = commit.Created
		build.Updated = commit.Updated
		err := createBuild(tx, rebind(stmtBuildInsert), build)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SetCommit updates an existing commit and commit tasks.
func (db *Commitstore) SetCommit(commit *types.Commit) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	commit.Updated = time.Now().UTC().Unix()
	err = updateCommit(tx, rebind(stmtCommitUpdate), commit)
	if err != nil {
		return err
	}

	for _, build := range commit.Builds {
		build.Updated = commit.Updated
		err = updateBuild(tx, rebind(stmtBuildUpdate), build)
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
WHERE commit_repo_id = ?
ORDER BY commit_sequence DESC
LIMIT ? OFFSET ?
`

// SQL query to retrieve the most recent commit.
// TODO exclude pull requests
const commitLastQuery = `
SELECT *
FROM commits
WHERE commit_repo_id = ?
  AND commit_branch  = ?
ORDER BY commit_sequence DESC
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
SELECT MAX(commit_sequence)
FROM commits
WHERE commit_repo_id = ?
`
