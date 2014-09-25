package database

import (
	"database/sql"
	"time"

	"github.com/drone/drone/shared/model"
	"github.com/russross/meddler"
)

type CommitManager interface {
	// Find finds the commit by ID.
	Find(id int64) (*model.Commit, error)

	// FindSha finds the commit for the branch and sha.
	FindSha(repo int64, branch, sha string) (*model.Commit, error)

	// FindLatest finds the most recent commit for the branch.
	FindLatest(repo int64, branch string) (*model.Commit, error)

	// FindOutput finds the commit's output.
	FindOutput(commit int64) ([]byte, error)

	// List finds recent commits for the repository
	List(repo int64) ([]*model.Commit, error)

	// ListBranch finds recent commits for the repository and branch.
	ListBranch(repo int64, branch string) ([]*model.Commit, error)

	// ListBranches finds most recent commit for each branch.
	//ListBranches(repo int64) ([]*model.Commit, error)

	// ListUser finds most recent commits for a user.
	ListUser(repo int64) ([]*model.CommitRepo, error)

	// Insert persists the commit to the datastore.
	Insert(commit *model.Commit) error

	// Update persists changes to the commit to the datastore.
	Update(commit *model.Commit) error

	// UpdateOutput persists a commit's stdout to the datastore.
	UpdateOutput(commit *model.Commit, out []byte) error

	// Delete removes the commit from the datastore.
	Delete(commit *model.Commit) error

	// CancelAll will update the status of all Started or Pending
	// builds to a status of Killed (cancelled).
	CancelAll() error
}

// commitManager manages a list of commits in a SQL database.
type commitManager struct {
	*sql.DB
}

// NewCommitManager initiales a new CommitManager intended to
// manage and persist commits.
func NewCommitManager(db *sql.DB) CommitManager {
	return &commitManager{db}
}

// SQL query to retrieve the latest Commits for each branch.
const listBranchesQuery = `
SELECT *
FROM commits
WHERE commit_id IN (
    SELECT MAX(commit_id)
    FROM commits
    WHERE repo_id=?
    AND commit_status NOT IN ('Started', 'Pending')
    GROUP BY commit_branch)
 ORDER BY commit_branch ASC
 `

// SQL query to retrieve the latest Commits for a specific branch.
const listBranchQuery = `
SELECT *
FROM commits
WHERE repo_id=?
AND   commit_branch=?
ORDER BY commit_id DESC
LIMIT 20
 `

// SQL query to retrieve the latest Commits for a user's repositories.
//const listUserCommitsQuery = `
//SELECT r.repo_remote, r.repo_host, r.repo_owner, r.repo_name, c.*
//FROM commits c, repos r, perms p
//WHERE c.repo_id=r.repo_id
//AND   r.repo_id=p.repo_id
//AND   p.user_id=?
//AND   c.commit_status NOT IN ('Started', 'Pending')
//ORDER BY commit_id DESC
//LIMIT 20
//`

const listUserCommitsQuery = `
SELECT r.repo_remote, r.repo_host, r.repo_owner, r.repo_name, c.*
FROM commits c, repos r
WHERE c.repo_id=r.repo_id
AND   c.commit_id IN (
	SELECT max(c.commit_id)
	FROM commits c, repos r, perms p
	WHERE c.repo_id=r.repo_id
	AND   r.repo_id=p.repo_id
	AND   p.user_id=?
	AND   c.commit_id
	AND   c.commit_status NOT IN ('Started', 'Pending')
	GROUP BY r.repo_id
) ORDER BY c.commit_created DESC LIMIT 5;
`

// SQL query to retrieve the latest Commits across all branches.
const listCommitsQuery = `
SELECT *
FROM commits
WHERE repo_id=? 
ORDER BY commit_id DESC
LIMIT 20
`

// SQL query to retrieve a Commit by branch and sha.
const findCommitQuery = `
SELECT *
FROM commits
WHERE repo_id=?
AND   commit_branch=?
AND   commit_sha=?
LIMIT 1
`

// SQL query to retrieve the most recent Commit for a branch.
const findLatestCommitQuery = `
SELECT *
FROM commits
WHERE commit_id IN (
    SELECT MAX(commit_id)
    FROM commits
    WHERE repo_id=?
    AND commit_branch=?)
`

// SQL query to retrieve a Commit's stdout.
const findOutputQuery = `
SELECT output_raw
FROM output
WHERE commit_id = ?
`

// SQL statement to insert a Commit's stdout.
const insertOutputStmt = `
INSERT INTO output (commit_id, output_raw) values (?,?);
`

// SQL statement to update a Commit's stdout.
const updateOutputStmt = `
UPDATE output SET output_raw = ? WHERE commit_id = ?;
`

// SQL statement to delete a Commit by ID.
const deleteCommitStmt = `
DELETE FROM commits WHERE commit_id = ?;
`

// SQL statement to cancel all running Commits.
const cancelCommitStmt = `
UPDATE commits SET
commit_status = ?,
commit_started = ?,
commit_finished = ?
WHERE commit_status IN ('Started', 'Pending');
`

func (db *commitManager) Find(id int64) (*model.Commit, error) {
	dst := model.Commit{}
	err := meddler.Load(db, "commits", &dst, id)
	return &dst, err
}

func (db *commitManager) FindSha(repo int64, branch, sha string) (*model.Commit, error) {
	dst := model.Commit{}
	err := meddler.QueryRow(db, &dst, findCommitQuery, repo, branch, sha)
	return &dst, err
}

func (db *commitManager) FindLatest(repo int64, branch string) (*model.Commit, error) {
	dst := model.Commit{}
	err := meddler.QueryRow(db, &dst, findLatestCommitQuery, repo, branch)
	return &dst, err
}

func (db *commitManager) FindOutput(commit int64) ([]byte, error) {
	var dst string
	err := db.QueryRow(findOutputQuery, commit).Scan(&dst)
	return []byte(dst), err
}

func (db *commitManager) List(repo int64) ([]*model.Commit, error) {
	var dst []*model.Commit
	err := meddler.QueryAll(db, &dst, listCommitsQuery, repo)
	return dst, err
}

func (db *commitManager) ListBranch(repo int64, branch string) ([]*model.Commit, error) {
	var dst []*model.Commit
	err := meddler.QueryAll(db, &dst, listBranchQuery, repo, branch)
	return dst, err
}

func (db *commitManager) ListBranches(repo int64) ([]*model.Commit, error) {
	var dst []*model.Commit
	err := meddler.QueryAll(db, &dst, listBranchesQuery, repo)
	return dst, err
}

func (db *commitManager) ListUser(user int64) ([]*model.CommitRepo, error) {
	var dst []*model.CommitRepo
	err := meddler.QueryAll(db, &dst, listUserCommitsQuery, user)
	return dst, err
}

func (db *commitManager) Insert(commit *model.Commit) error {
	commit.Created = time.Now().Unix()
	commit.Updated = time.Now().Unix()
	return meddler.Insert(db, "commits", commit)
}

func (db *commitManager) Update(commit *model.Commit) error {
	commit.Updated = time.Now().Unix()
	return meddler.Update(db, "commits", commit)
}

func (db *commitManager) UpdateOutput(commit *model.Commit, out []byte) error {
	_, err := db.Exec(insertOutputStmt, commit.ID, out)
	if err != nil {
		return nil
	}
	_, err = db.Exec(updateOutputStmt, out, commit.ID)
	return err
}

func (db *commitManager) Delete(commit *model.Commit) error {
	_, err := db.Exec(deleteCommitStmt, commit.ID)
	return err
}

func (db *commitManager) CancelAll() error {
	_, err := db.Exec(cancelCommitStmt, model.StatusKilled, time.Now().Unix(), time.Now().Unix())
	return err
}
