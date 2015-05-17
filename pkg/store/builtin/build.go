package builtin

import (
	"time"

	common "github.com/drone/drone/pkg/types"
	"github.com/russross/meddler"
)

type Buildstore struct {
	meddler.DB
}

func NewBuildstore(db meddler.DB) *Buildstore {
	return &Buildstore{db}
}

// Build returns a build by ID.
func (db *Buildstore) Build(id int64) (*common.Build, error) {
	var build = new(common.Build)
	var err = meddler.Load(db, buildTable, build, id)
	return build, err
}

// BuildSeq returns a build by sequence number.
func (db *Buildstore) BuildSeq(commit *common.Commit, seq int) (*common.Build, error) {
	var build = new(common.Build)
	var err = meddler.QueryRow(db, build, rebind(buildNumberQuery), commit.ID, seq)
	return build, err
}

// BuildList returns a list of all commit builds
func (db *Buildstore) BuildList(commit *common.Commit) ([]*common.Build, error) {
	var builds []*common.Build
	var err = meddler.QueryAll(db, &builds, rebind(buildListQuery), commit.ID)
	return builds, err
}

// SetBuild updates an existing build.
func (db *Buildstore) SetBuild(build *common.Build) error {
	build.Updated = time.Now().UTC().Unix()
	return meddler.Update(db, buildTable, build)
}

// Build table name in database.
const buildTable = "builds"

// SQL query to retrieve a token by label.
const buildListQuery = `
SELECT *
FROM builds
WHERE commit_id = ?
ORDER BY build_seq ASC
`

const buildNumberQuery = `
SELECT *
FROM builds
WHERE commit_id = ?
  AND build_seq = ?
LIMIT 1;
`
