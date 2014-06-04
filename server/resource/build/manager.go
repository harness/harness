package build

import (
	"database/sql"
	"time"

	"github.com/russross/meddler"
)

type BuildManager interface {
	// Find finds the build by ID.
	Find(id int64) (*Build, error)

	// FindNumber finds the build with the specified build number.
	FindNumber(commit, number int64) (*Build, error)

	// FindOutput finds the build's output.
	FindOutput(commit, number int64) ([]byte, error)

	// List finds all builds for the commit ID.
	List(commit int64) ([]*Build, error)

	// Insert persists the build to the datastore.
	Insert(build *Build) error

	// Update persists changes to the build to the datastore.
	Update(build *Build) error

	// UpdateOutput persists a build's stdout to the datastore.
	UpdateOutput(build *Build, out []byte) error

	// Delete removes the build from the datastore.
	Delete(build *Build) error
}

// buildManager manages a list of builds in a SQL database.
type buildManager struct {
	*sql.DB
}

// SQL query to retrieve a list of builds by commit ID.
const listQuery = `
SELECT build_id, commit_id, build_number, build_matrix, build_status,
build_started, build_finished, build_duration, build_created, build_updated
FROM builds
WHERE commit_id=? 
ORDER BY build_number ASC
`

// SQL query to retrieve a build by commit ID and build number.
const findQuery = `
SELECT build_id, commit_id, build_number, build_matrix, build_status,
build_started, build_finished, build_duration, build_created, build_updated
FROM builds
WHERE commit_id=?
AND   build_number=?
LIMIT 1
`

// SQL query to retrieve a build's console output by build ID.
const findOutputQuery = `
SELECT build_console
FROM builds
WHERE commit_id=?
AND   build_number=?
LIMIT 1
`

// SQL statement to update a build's console output.
const updateStmt = `
UPDATE builds set build_console=? WHERE build_id=?
`

// SQL statement to delete a build by ID.
const deleteStmt = `
DELETE FROM builds WHERE build_id = ?
`

// NewManager initiales a new BuildManager intended to
// manage and persist builds.
func NewManager(db *sql.DB) BuildManager {
	return &buildManager{db}
}

func (db *buildManager) Find(id int64) (*Build, error) {
	dst := Build{}
	err := meddler.Load(db, "builds", &dst, id)
	return &dst, err
}

func (db *buildManager) FindNumber(commit, number int64) (*Build, error) {
	dst := Build{}
	err := meddler.QueryRow(db, &dst, findQuery, commit, number)
	return &dst, err
}

func (db *buildManager) FindOutput(commit, number int64) ([]byte, error) {
	var dst string
	err := db.QueryRow(findOutputQuery, commit, number).Scan(&dst)
	return []byte(dst), err
}

func (db *buildManager) List(commit int64) ([]*Build, error) {
	var dst []*Build
	err := meddler.QueryAll(db, &dst, listQuery, commit)
	return dst, err
}

func (db *buildManager) Insert(build *Build) error {
	build.Created = time.Now().Unix()
	build.Updated = time.Now().Unix()
	return meddler.Insert(db, "builds", build)
}

func (db *buildManager) Update(build *Build) error {
	build.Updated = time.Now().Unix()
	return meddler.Update(db, "builds", build)
}

func (db *buildManager) UpdateOutput(build *Build, out []byte) error {
	_, err := db.Exec(updateStmt, out, build.ID)
	return err
}

func (db *buildManager) Delete(build *Build) error {
	_, err := db.Exec(deleteStmt, build.ID)
	return err
}
