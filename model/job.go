package model

import (
	"github.com/drone/drone/shared/database"
	"github.com/russross/meddler"
)

type Job struct {
	ID       int64  `json:"id"           meddler:"job_id,pk"`
	BuildID  int64  `json:"-"            meddler:"job_build_id"`
	NodeID   int64  `json:"-"            meddler:"job_node_id"`
	Number   int    `json:"number"       meddler:"job_number"`
	Status   string `json:"status"       meddler:"job_status"`
	ExitCode int    `json:"exit_code"    meddler:"job_exit_code"`
	Enqueued int64  `json:"enqueued_at"  meddler:"job_enqueued"`
	Started  int64  `json:"started_at"   meddler:"job_started"`
	Finished int64  `json:"finished_at"  meddler:"job_finished"`

	Environment map[string]string `json:"environment" meddler:"job_environment,json"`
}

func GetJob(db meddler.DB, id int64) (*Job, error) {
	var job = new(Job)
	var err = meddler.Load(db, jobTable, job, id)
	return job, err
}

func GetJobNumber(db meddler.DB, build *Build, number int) (*Job, error) {
	var job = new(Job)
	var err = meddler.QueryRow(db, job, database.Rebind(jobNumberQuery), build.ID, number)
	return job, err
}

func GetJobList(db meddler.DB, build *Build) ([]*Job, error) {
	var jobs = []*Job{}
	var err = meddler.QueryAll(db, &jobs, database.Rebind(jobListQuery), build.ID)
	return jobs, err
}

func InsertJob(db meddler.DB, job *Job) error {
	return meddler.Insert(db, jobTable, job)
}

func UpdateJob(db meddler.DB, job *Job) error {
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
