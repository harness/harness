package datastore

import (
	"bytes"
	"database/sql"
	"io"
	"io/ioutil"

	"github.com/drone/drone/model"
	"github.com/russross/meddler"
)

type logstore struct {
	*sql.DB
}

func (db *logstore) Read(job *model.Job) (io.ReadCloser, error) {
	var log = new(model.Log)
	var err = meddler.QueryRow(db, log, rebind(logQuery), job.ID)
	var buf = bytes.NewBuffer(log.Data)
	return ioutil.NopCloser(buf), err
}

func (db *logstore) Write(job *model.Job, r io.Reader) error {
	var log = new(model.Log)
	var err = meddler.QueryRow(db, log, rebind(logQuery), job.ID)
	if err != nil {
		log = &model.Log{JobID: job.ID}
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
