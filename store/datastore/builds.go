package datastore

import (
	"time"

	"github.com/drone/drone/model"
	"github.com/russross/meddler"
)

func (db *datastore) GetBuild(id int64) (*model.Build, error) {
	var build = new(model.Build)
	var err = meddler.Load(db, buildTable, build, id)
	return build, err
}

func (db *datastore) GetBuildNumber(repo *model.Repo, num int) (*model.Build, error) {
	var build = new(model.Build)
	var err = meddler.QueryRow(db, build, rebind(buildNumberQuery), repo.ID, num)
	return build, err
}

func (db *datastore) GetBuildRef(repo *model.Repo, ref string) (*model.Build, error) {
	var build = new(model.Build)
	var err = meddler.QueryRow(db, build, rebind(buildRefQuery), repo.ID, ref)
	return build, err
}

func (db *datastore) GetBuildCommit(repo *model.Repo, sha, branch string) (*model.Build, error) {
	var build = new(model.Build)
	var err = meddler.QueryRow(db, build, rebind(buildCommitQuery), repo.ID, sha, branch)
	return build, err
}

func (db *datastore) GetBuildLast(repo *model.Repo, branch string) (*model.Build, error) {
	var build = new(model.Build)
	var err = meddler.QueryRow(db, build, rebind(buildLastQuery), repo.ID, branch)
	return build, err
}

func (db *datastore) GetBuildLastBefore(repo *model.Repo, branch string, num int64) (*model.Build, error) {
	var build = new(model.Build)
	var err = meddler.QueryRow(db, build, rebind(buildLastBeforeQuery), repo.ID, branch, num)
	return build, err
}

func (db *datastore) GetBuildList(repo *model.Repo) ([]*model.Build, error) {
	var builds = []*model.Build{}
	var err = meddler.QueryAll(db, &builds, rebind(buildListQuery), repo.ID)
	return builds, err
}

func (db *datastore) GetBuildQueue() ([]*model.Feed, error) {
	feed := []*model.Feed{}
	err := meddler.QueryAll(db, &feed, buildQueueList)
	return feed, err
}

func (db *datastore) CreateBuild(build *model.Build, jobs ...*model.Job) error {
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

func (db *datastore) UpdateBuild(build *model.Build) error {
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

const buildQueueList = `
SELECT
 repo_owner
,repo_name
,repo_full_name
,build_number
,build_event
,build_status
,build_created
,build_started
,build_finished
,build_commit
,build_branch
,build_ref
,build_refspec
,build_remote
,build_title
,build_message
,build_author
,build_email
,build_avatar
FROM
 builds b
,repos r
WHERE b.build_repo_id = r.repo_id
  AND b.build_status IN ('pending','running')
`
