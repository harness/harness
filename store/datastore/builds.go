package datastore

import (
	"database/sql"
	"time"

	"github.com/drone/drone/model"
	"github.com/russross/meddler"
)

type buildstore struct {
	*sql.DB
}

func (db *buildstore) Get(id int64) (*model.Build, error) {
	var build = new(model.Build)
	var err = meddler.Load(db, buildTable, build, id)
	return build, err
}

func (db *buildstore) GetNumber(repo *model.Repo, num int) (*model.Build, error) {
	var build = new(model.Build)
	var err = meddler.QueryRow(db, build, rebind(buildNumberQuery), repo.ID, num)
	return build, err
}

func (db *buildstore) GetRef(repo *model.Repo, ref string) (*model.Build, error) {
	var build = new(model.Build)
	var err = meddler.QueryRow(db, build, rebind(buildRefQuery), repo.ID, ref)
	return build, err
}

func (db *buildstore) GetCommit(repo *model.Repo, sha, branch string) (*model.Build, error) {
	var build = new(model.Build)
	var err = meddler.QueryRow(db, build, rebind(buildCommitQuery), repo.ID, sha, branch)
	return build, err
}

func (db *buildstore) GetLast(repo *model.Repo, branch string) (*model.Build, error) {
	var build = new(model.Build)
	var err = meddler.QueryRow(db, build, rebind(buildLastQuery), repo.ID, branch)
	return build, err
}

func (db *buildstore) GetLastBefore(repo *model.Repo, branch string, num int64) (*model.Build, error) {
	var build = new(model.Build)
	var err = meddler.QueryRow(db, build, rebind(buildLastBeforeQuery), repo.ID, branch, num)
	return build, err
}

func (db *buildstore) GetList(repo *model.Repo) ([]*model.Build, error) {
	var builds = []*model.Build{}
	var err = meddler.QueryAll(db, &builds, rebind(buildListQuery), repo.ID)
	return builds, err
}

func (db *buildstore) Create(build *model.Build, jobs ...*model.Job) error {
	var number int
	db.QueryRow(rebind(buildNumberLast), build.RepoID).Scan(&number)
	build.Number = number + 1
	build.Created = time.Now().UTC().Unix()
	build.Enqueued = build.Created
	err := meddler.Insert(db, buildTable, build)
	if err != nil {
		return err
	}
	for i, job := range jobs {
		job.BuildID = build.ID
		job.Number = i + 1
		job.Enqueued = build.Created
		err = meddler.Insert(db, jobTable, job)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *buildstore) Update(build *model.Build) error {
	return meddler.Update(db, buildTable, build)
}

const buildTable = "builds"

const buildListQuery = `
SELECT *
FROM builds
WHERE build_repo_id = ?
ORDER BY build_number DESC
LIMIT 50
`

const buildNumberQuery = `
SELECT *
FROM builds
WHERE build_repo_id = ?
  AND build_number = ?
LIMIT 1;
`

const buildLastQuery = `
SELECT *
FROM builds
WHERE build_repo_id = ?
  AND build_branch  = ?
  AND build_event   = 'push'
ORDER BY build_number DESC
LIMIT 1
`

const buildLastBeforeQuery = `
SELECT *
FROM builds
WHERE build_repo_id = ?
  AND build_branch  = ?
  AND build_id < ?
ORDER BY build_number DESC
LIMIT 1
`

const buildCommitQuery = `
SELECT *
FROM builds
WHERE build_repo_id = ?
  AND build_commit  = ?
  AND build_branch  = ?
LIMIT 1
`

const buildRefQuery = `
SELECT *
FROM builds
WHERE build_repo_id = ?
  AND build_ref     = ?
LIMIT 1
`

const buildNumberLast = `
SELECT MAX(build_number)
FROM builds
WHERE build_repo_id = ?
`
