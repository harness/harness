package model

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/drone/drone/shared/database"
	"github.com/russross/meddler"
)

type Log struct {
	ID    int64  `meddler:"log_id,pk"`
	JobID int64  `meddler:"log_job_id"`
	Data  []byte `meddler:"log_data"`
}

func GetLog(db meddler.DB, job *Job) (io.ReadCloser, error) {
	var log = new(Log)
	var err = meddler.QueryRow(db, log, database.Rebind(logQuery), job.ID)
	var buf = bytes.NewBuffer(log.Data)
	return ioutil.NopCloser(buf), err
}

func SetLog(db meddler.DB, job *Job, r io.Reader) error {
	var log = new(Log)
	var err = meddler.QueryRow(db, log, database.Rebind(logQuery), job.ID)
	if err != nil {
		log = &Log{JobID: job.ID}
	}
	log.Data, _ = ioutil.ReadAll(r)
	return meddler.Save(db, logTable, log)
}

const logTable = "logs"

const logQuery = `
SELECT *
FROM logs
WHERE log_job_id=?
LIMIT 1
`
