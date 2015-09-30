package model

import (
	"time"

	"github.com/drone/drone/shared/database"
	"github.com/russross/meddler"
)

type Build struct {
	ID        int64  `json:"id"            meddler:"build_id,pk"`
	RepoID    int64  `json:"-"             meddler:"build_repo_id"`
	Number    int    `json:"number"        meddler:"build_number"`
	Event     string `json:"event"         meddler:"build_event"`
	Status    string `json:"status"        meddler:"build_status"`
	Created   int64  `json:"created_at"    meddler:"build_created"`
	Started   int64  `json:"started_at"    meddler:"build_started"`
	Finished  int64  `json:"finished_at"   meddler:"build_finished"`
	Commit    string `json:"commit"        meddler:"build_commit"`
	Branch    string `json:"branch"        meddler:"build_branch"`
	Ref       string `json:"ref"           meddler:"build_ref"`
	Refspec   string `json:"refspec"       meddler:"build_refspec"`
	Remote    string `json:"remote"        meddler:"build_remote"`
	Title     string `json:"title"         meddler:"build_title"`
	Message   string `json:"message"       meddler:"build_message"`
	Timestamp string `json:"timestamp"     meddler:"build_timestamp"`
	Author    string `json:"author"        meddler:"build_author"`
	Avatar    string `json:"author_avatar" meddler:"build_avatar"`
	Email     string `json:"author_email"  meddler:"build_email"`
	Link      string `json:"link_url"      meddler:"build_link"`
}

type BuildGroup struct {
	Date   string
	Builds []*Build
}

func GetBuild(db meddler.DB, id int64) (*Build, error) {
	var build = new(Build)
	var err = meddler.Load(db, buildTable, build, id)
	return build, err
}

func GetBuildNumber(db meddler.DB, repo *Repo, number int) (*Build, error) {
	var build = new(Build)
	var err = meddler.QueryRow(db, build, database.Rebind(buildNumberQuery), repo.ID, number)
	return build, err
}

func GetBuildRef(db meddler.DB, repo *Repo, ref string) (*Build, error) {
	var build = new(Build)
	var err = meddler.QueryRow(db, build, database.Rebind(buildRefQuery), repo.ID, ref)
	return build, err
}

func GetBuildCommit(db meddler.DB, repo *Repo, sha, branch string) (*Build, error) {
	var build = new(Build)
	var err = meddler.QueryRow(db, build, database.Rebind(buildCommitQuery), repo.ID, sha, branch)
	return build, err
}

func GetBuildLast(db meddler.DB, repo *Repo, branch string) (*Build, error) {
	var build = new(Build)
	var err = meddler.QueryRow(db, build, database.Rebind(buildLastQuery), repo.ID, branch)
	return build, err
}

func GetBuildList(db meddler.DB, repo *Repo) ([]*Build, error) {
	var builds = []*Build{}
	var err = meddler.QueryAll(db, &builds, database.Rebind(buildListQuery), repo.ID)
	return builds, err
}

func CreateBuild(db meddler.DB, build *Build, jobs ...*Job) error {
	var number int
	db.QueryRow(buildNumberLast, build.RepoID).Scan(&number)
	build.Number = number + 1
	build.Created = time.Now().UTC().Unix()
	err := meddler.Insert(db, buildTable, build)
	if err != nil {
		return err
	}
	for i, job := range jobs {
		job.BuildID = build.ID
		job.Number = i + 1
		err = InsertJob(db, job)
		if err != nil {
			return err
		}
	}
	return nil
}

func UpdateBuild(db meddler.DB, build *Build) error {
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
