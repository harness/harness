package builtin

import (
	"database/sql"

	"github.com/drone/drone/pkg/types"
)

type Buildstore struct {
	*sql.DB
}

func NewBuildstore(db *sql.DB) *Buildstore {
	return &Buildstore{db}
}

// Build gets a build by ID
func (db *Buildstore) Build(id int64) (*types.Build, error) {
	return getBuild(db, rebind(stmtBuildSelect), id)
}

// BuildNumber gets the specified build number for the
// named repository and build number
func (db *Buildstore) BuildNumber(repo *types.Repo, seq int) (*types.Build, error) {
	return getBuild(db, rebind(stmtBuildSelectBuildNumber), repo.ID, seq)
}

// BuildPullRequest gets the specific build for the
// named repository and pull request number
func (db *Buildstore) BuildPullRequestNumber(repo *types.Repo, seq int) (*types.Build, error) {
	return getBuild(db, rebind(stmtBuildSelectPullRequestNumber), repo.ID, seq)
}

// BuildSha gets the specific build for the
// named repository and sha
func (db *Buildstore) BuildSha(repo *types.Repo, sha, branch string) (*types.Build, error) {
	return getBuild(db, rebind(stmtBuildSelectSha), repo.ID, sha, branch)
}

// BuildLast gets the last executed build for the
// named repository.
func (db *Buildstore) BuildLast(repo *types.Repo, branch string) (*types.Build, error) {
	return getBuild(db, rebind(buildLastQuery), repo.ID, branch)
}

// BuildList gets a list of recent builds for the
// named repository.
func (db *Buildstore) BuildList(repo *types.Repo, limit, offset int) ([]*types.Build, error) {
	return getBuilds(db, rebind(buildListQuery), repo.ID, limit, offset)
}

// AddBuild inserts a new build in the datastore.
func (db *Buildstore) AddBuild(build *types.Build) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// extract the next build number from the database
	row := tx.QueryRow(rebind(buildNumberLast), build.RepoID)
	if row != nil {
		row.Scan(&build.Number)
	}

	build.Number = build.Number + 1 // increment
	err = createBuild(tx, rebind(stmtBuildInsert), build)
	if err != nil {
		return err
	}

	for _, job := range build.Jobs {
		job.BuildID = build.ID
		err := createJob(tx, rebind(stmtJobInsert), job)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SetBuild updates an existing build and build jobs.
func (db *Buildstore) SetBuild(build *types.Build) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = updateBuild(tx, rebind(stmtBuildUpdate), build)
	if err != nil {
		return err
	}

	for _, job := range build.Jobs {
		err = updateJob(tx, rebind(stmtJobUpdate), job)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// KillBuilds updates all pending or started builds
// in the datastore settings the status to killed.
func (db *Buildstore) KillBuilds() error {
	var _, err1 = db.Exec(rebind(buildKillStmt))
	if err1 != nil {
		return err1
	}
	var _, err2 = db.Exec(rebind(jobKillStmt))
	return err2
}

const stmtBuildSelectPullRequestNumber = stmtBuildSelectList + `
WHERE build_repo_id = ?
AND build_pull_request_number = ?
ORDER BY build_number DESC
LIMIT 1
`

const stmtBuildSelectSha = stmtBuildSelectList + `
WHERE build_repo_id = ?
AND build_commit_sha = ?
AND build_commit_branch = ?
ORDER BY build_number DESC
LIMIT 1
`

// SQL query to retrieve the latest builds across all branches.
const buildListQuery = `
SELECT
 build_id
,build_repo_id
,build_number
,build_status
,build_started
,build_finished
,build_commit_sha
,build_commit_ref
,build_commit_link
,build_commit_branch
,build_commit_message
,build_commit_timestamp
,build_commit_remote
,build_commit_author_login
,build_commit_author_email
,build_pull_request_number
,build_pull_request_title
,build_pull_request_link
,build_pull_request_base_sha
,build_pull_request_base_ref
,build_pull_request_base_link
,build_pull_request_base_branch
,build_pull_request_base_message
,build_pull_request_base_timestamp
,build_pull_request_base_remote
,build_pull_request_base_author_login
,build_pull_request_base_author_email
FROM builds
WHERE build_repo_id = ?
ORDER BY build_number DESC
LIMIT ? OFFSET ?
`

// SQL query to retrieve the most recent build.
// TODO exclude pull requests
const buildLastQuery = `
SELECT
 build_id
,build_repo_id
,build_number
,build_status
,build_started
,build_finished
,build_commit_sha
,build_commit_ref
,build_commit_link
,build_commit_branch
,build_commit_message
,build_commit_timestamp
,build_commit_remote
,build_commit_author_login
,build_commit_author_email
,build_pull_request_number
,build_pull_request_title
,build_pull_request_link
,build_pull_request_base_sha
,build_pull_request_base_ref
,build_pull_request_base_link
,build_pull_request_base_branch
,build_pull_request_base_message
,build_pull_request_base_timestamp
,build_pull_request_base_remote
,build_pull_request_base_author_login
,build_pull_request_base_author_email
FROM builds
WHERE build_repo_id = ?
  AND build_commit_branch  = ?
ORDER BY build_number DESC
LIMIT 1
`

// SQL statement to cancel all running builds.
const buildKillStmt = `
UPDATE builds SET build_status = 'killed'
WHERE build_status IN ('pending', 'running');
`

// SQL statement to cancel all running build jobs.
const jobKillStmt = `
UPDATE jobs SET job_status = 'killed'
WHERE job_status IN ('pending', 'running');
`

// SQL statement to retrieve the latest sequential
// build number for a build
const buildNumberLast = `
SELECT MAX(build_number)
FROM builds
WHERE build_repo_id = ?
`
