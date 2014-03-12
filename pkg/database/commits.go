package database

import (
	"time"

	. "github.com/drone/drone/pkg/model"
	"github.com/russross/meddler"
)

// Name of the Commit table in the database
const commitTable = "commits"

// SQL Queries to retrieve a list of all Commits belonging to a Repo.
const commitStmt = `
SELECT id, repo_id, status, started, finished, duration,
hash, branch, pull_request, author, gravatar, timestamp, message, created, updated
FROM commits
WHERE repo_id = ? AND branch = ?
ORDER BY created DESC
LIMIT 10
`

// SQL Queries to retrieve the latest Commit.
const commitLatestStmt = `
SELECT id, repo_id, status, started, finished, duration,
hash, branch, pull_request, author, gravatar, timestamp, message, created, updated
FROM commits
WHERE repo_id = ? AND branch = ?
ORDER BY created DESC
LIMIT 1
`

// SQL Queries to retrieve a Commit by id.
const commitFindStmt = `
SELECT id, repo_id, status, started, finished, duration,
hash, branch, pull_request, author, gravatar, timestamp, message, created, updated
FROM commits
WHERE id = ?
`

// SQL Queries to retrieve a Commit by name and repo id.
const commitFindHashStmt = `
SELECT id, repo_id, status, started, finished, duration,
hash, branch, pull_request, author, gravatar, timestamp, message, created, updated
FROM commits
WHERE hash = ? AND repo_id = ?
LIMIT 1
`

// SQL Query to retrieve a list of recent commits by user.
const userCommitRecentStmt = `
SELECT r.slug, r.host, r.owner, r.name,
c.status, c.started, c.finished, c.duration, c.hash, c.branch, c.pull_request,
c.author, c.gravatar, c.timestamp, c.message, c.created, c.updated
FROM repos r, commits c
WHERE r.user_id = ?
AND   r.team_id = 0
AND   r.id = c.repo_id
AND   c.status IN ('Success', 'Failure')
ORDER BY c.created desc
LIMIT 10
`

// SQL Query to retrieve a list of recent commits by team.
const teamCommitRecentStmt = `
SELECT r.slug, r.host, r.owner, r.name,
c.status, c.started, c.finished, c.duration, c.hash, c.branch, c.pull_request,
c.author, c.gravatar, c.timestamp, c.message, c.created, c.updated
FROM repos r, commits c
WHERE r.team_id = ?
AND   r.id = c.repo_id
AND   c.status IN ('Success', 'Failure')
ORDER BY c.created desc
LIMIT 10
`

// SQL Queries to delete a Commit.
const commitDeleteStmt = `
DELETE FROM commits WHERE id = ?
`

// SQL Queries to retrieve the latest Commits for each branch.
const commitBranchesStmt = `
SELECT id, repo_id, status, started, finished, duration,
hash, branch, pull_request, author, gravatar, timestamp, message, created, updated
FROM commits
WHERE id IN (
    SELECT MAX(id)
    FROM commits
    WHERE repo_id = ?
    GROUP BY branch)
 ORDER BY branch ASC
 `

// SQL Queries to retrieve the latest Commits for each branch.
const commitBranchStmt = `
SELECT id, repo_id, status, started, finished, duration,
hash, branch, pull_request, author, gravatar, timestamp, message, created, updated
FROM commits
WHERE id IN (
    SELECT MAX(id)
    FROM commits
    WHERE repo_id = ?
    AND   branch  = ?
    GROUP BY branch)
LIMIT 1
 `

// SQL Queries to fail all commits that are currently building
const commitFailStartedStmt = `
UPDATE commits
SET status = 'Failure'
WHERE status = 'Started'
`

// Returns the Commit with the given ID.
func GetCommit(id int64) (*Commit, error) {
	commit := Commit{}
	err := meddler.QueryRow(db, &commit, commitFindStmt, id)
	return &commit, err
}

// Returns the Commit with the given hash.
func GetCommitHash(hash string, repo int64) (*Commit, error) {
	commit := Commit{}
	err := meddler.QueryRow(db, &commit, commitFindHashStmt, hash, repo)
	return &commit, err
}

// Returns the most recent Commit for the given branch.
func GetBranch(repo int64, branch string) (*Commit, error) {
	commit := Commit{}
	err := meddler.QueryRow(db, &commit, commitBranchStmt, repo, branch)
	return &commit, err
}

// Creates a new Commit.
func SaveCommit(commit *Commit) error {
	if commit.ID == 0 {
		commit.Created = time.Now().UTC()
	}
	commit.Updated = time.Now().UTC()
	return meddler.Save(db, commitTable, commit)
}

// Deletes an existing Commit.
func DeleteCommit(id int64) error {
	_, err := db.Exec(commitDeleteStmt, id)
	return err
}

// Returns a list of all Commits associated
// with the specified Repo ID.
func ListCommits(repo int64, branch string) ([]*Commit, error) {
	var commits []*Commit
	err := meddler.QueryAll(db, &commits, commitStmt, repo, branch)
	return commits, err
}

// Returns a list of recent Commits associated
// with the specified User ID
func ListCommitsUser(user int64) ([]*RepoCommit, error) {
	var commits []*RepoCommit
	err := meddler.QueryAll(db, &commits, userCommitRecentStmt, user)
	return commits, err
}

// Returns a list of recent Commits associated
// with the specified Team ID
func ListCommitsTeam(team int64) ([]*RepoCommit, error) {
	var commits []*RepoCommit
	err := meddler.QueryAll(db, &commits, teamCommitRecentStmt, team)
	return commits, err
}

// Returns a list of the most recent commits for each branch.
func ListBranches(repo int64) ([]*Commit, error) {
	var commits []*Commit
	err := meddler.QueryAll(db, &commits, commitBranchesStmt, repo)
	return commits, err
}

func FailStartedCommits() error {
	_, err := db.Exec(commitFailStartedStmt)
	return err
}
