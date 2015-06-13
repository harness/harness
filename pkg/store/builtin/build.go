package builtin

import (
	"database/sql"
	"time"

	"github.com/drone/drone/pkg/types"
)

type Buildstore struct {
	*sql.DB
}

func NewBuildstore(db *sql.DB) *Buildstore {
	return &Buildstore{db}
}

// Build returns a build by ID.
func (db *Buildstore) Build(id int64) (*types.Build, error) {
	return getBuild(db, rebind(stmtBuildSelect), id)
}

// BuildSeq returns a build by sequence number.
func (db *Buildstore) BuildSeq(commit *types.Commit, seq int) (*types.Build, error) {
	return getBuild(db, rebind(stmtBuildSelectBuildSeq), commit.ID, seq)
}

// BuildList returns a list of all commit builds
func (db *Buildstore) BuildList(commit *types.Commit) ([]*types.Build, error) {
	return getBuilds(db, rebind(stmtBuildSelectBuildCommitId), commit.ID)
}

// SetBuild updates an existing build.
func (db *Buildstore) SetBuild(build *types.Build) error {
	build.Updated = time.Now().UTC().Unix()
	return updateBuild(db, rebind(stmtBuildUpdate), build)
}

// AddBuild inserts a build.
func (db *Buildstore) AddBuild(build *types.Build) error {
	build.Created = time.Now().UTC().Unix()
	build.Updated = time.Now().UTC().Unix()
	return createBuild(db, rebind(stmtBuildInsert), build)
}

// Build table name in database.
const buildTable = "builds"
