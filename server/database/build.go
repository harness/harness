package database

import (
	"database/sql"
	"time"

	"github.com/drone/drone/shared/model"
	"github.com/russross/meddler"
)

type BuildManager interface {
	Find(index, commit int64) (*model.Build, error)
	FindCommit(id int64) ([]*model.Build, error)
	Update(build *model.Build) error
	Insert(build *model.Build) error
	FindOutput(build int64) ([]byte, error)
	UpdateOutput(build *model.Build, out []byte) error
}

type buildManager struct {
	*sql.DB
}

// SQL query to retrieve a Build by index and commit_id.
const findBuildQuery = `
SELECT *
FROM builds
WHERE build_index=?
AND   commit_id=?
LIMIT 1
`

const listBuildsCommitQuery = `
SELECT *
FROM builds
WHERE commit_id=? 
ORDER BY build_id DESC
`

const findOutputQuery = `
SELECT build_output
FROM builds
WHERE build_id=?
`

// SQL statement to update a Commit's stdout.
const updateOutputStmt = `
UPDATE builds SET build_output = ? WHERE build_id = ?;
`

func NewBuildManager(db *sql.DB) BuildManager {
	return &buildManager{db}
}

func (db *buildManager) Find(index, commit int64) (*model.Build, error) {
	dst := model.Build{}
	err := meddler.QueryRow(db, &dst, findBuildQuery, index, commit)
	return &dst, err
}

func (db *buildManager) FindCommit(id int64) ([]*model.Build, error) {
	var dst []*model.Build
	err := meddler.QueryAll(db, &dst, listBuildsCommitQuery, id)
	return dst, err
}

func (db *buildManager) FindOutput(build int64) ([]byte, error) {
	var dst string
	err := db.QueryRow(findOutputQuery, build).Scan(&dst)
	return []byte(dst), err
}

func (db *buildManager) Update(build *model.Build) error {
	build.Updated = time.Now().Unix()
	return meddler.Update(db, "builds", build)
}

func (db *buildManager) UpdateOutput(build *model.Build, out []byte) error {
	_, err := db.Exec(updateOutputStmt, out, build.ID)
	return err
}

func (db *buildManager) Insert(build *model.Build) error {
	build.Created = time.Now().Unix()
	build.Updated = time.Now().Unix()
	return meddler.Insert(db, "builds", build)
}
