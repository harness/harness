package datastore

import (
	"database/sql"

	"github.com/drone/drone/model"
	"github.com/russross/meddler"
)

type jobstore struct {
	*sql.DB
}

func (db *jobstore) Get(id int64) (*model.Job, error) {
	var job = new(model.Job)
	var err = meddler.Load(db, jobTable, job, id)
	return job, err
}

func (db *jobstore) GetNumber(build *model.Build, num int) (*model.Job, error) {
	var job = new(model.Job)
	var err = meddler.QueryRow(db, job, rebind(jobNumberQuery), build.ID, num)
	return job, err
}

func (db *jobstore) GetList(build *model.Build) ([]*model.Job, error) {
	var jobs = []*model.Job{}
	var err = meddler.QueryAll(db, &jobs, rebind(jobListQuery), build.ID)
	return jobs, err
}

func (db *jobstore) Create(job *model.Job) error {
	return meddler.Insert(db, jobTable, job)
}

func (db *jobstore) Update(job *model.Job) error {
	return meddler.Update(db, jobTable, job)
}

const jobTable = "jobs"

const jobListQuery = `
SELECT *
FROM jobs
WHERE job_build_id = ?
ORDER BY job_number ASC
`

const jobNumberQuery = `
SELECT *
FROM jobs
WHERE job_build_id = ?
AND   job_number = ?
LIMIT 1
`
