package database

import (
	. "github.com/drone/drone/pkg/model"
	"github.com/russross/meddler"
)

// Name of the Build table in the database
const buildTable = "builds"

// SQL Queries to retrieve a list of all Commits belonging to a Repo.
const buildStmt = `
SELECT id, commit_id, slug, status, started, finished, duration, created, updated, stdout, buildscript
FROM builds
WHERE commit_id = ?
ORDER BY slug ASC
`

// SQL Queries to retrieve a Build by id.
const buildFindStmt = `
SELECT id, commit_id, slug, status, started, finished, duration, created, updated, stdout, buildscript
FROM builds
WHERE id = ?
LIMIT 1
`

// SQL Queries to retrieve a Commit by name and repo id.
const buildFindSlugStmt = `
SELECT id, commit_id, slug, status, started, finished, duration, created, updated, stdout, buildscript
FROM builds
WHERE slug = ? AND commit_id = ?
LIMIT 1
`

// SQL Queries to fail all builds that are running or pending
const buildFailUnfinishedStmt = `
UPDATE builds
SET status = 'Failure'
WHERE status IN ('Started', 'Pending')
`

// SQL Queries to delete a Commit.
const buildDeleteStmt = `
DELETE FROM builds WHERE id = ?
`

// Returns the Build with the given ID.
func GetBuild(id int64) (*Build, error) {
	build := Build{}
	err := meddler.QueryRow(db, &build, buildFindStmt, id)
	return &build, err
}

// Returns the Build with the given slug.
func GetBuildSlug(slug string, commit int64) (*Build, error) {
	build := Build{}
	err := meddler.QueryRow(db, &build, buildFindSlugStmt, slug, commit)
	return &build, err
}

// Creates a new Build.
func SaveBuild(build *Build) error {
	return meddler.Save(db, buildTable, build)
}

// Deletes an existing Build.
func DeleteBuild(id int64) error {
	_, err := db.Exec(buildDeleteStmt, id)
	return err
}

// Returns a list of all Builds associated
// with the specified Commit ID and branch.
func ListBuilds(id int64) ([]*Build, error) {
	var builds []*Build
	err := meddler.QueryAll(db, &builds, buildStmt, id)
	return builds, err
}

// FailUnfinishedBuilds sets status=Failure to all builds
// in the Pending and Started states
func FailUnfinishedBuilds() error {
	_, err := db.Exec(buildFailUnfinishedStmt)
	return err
}
