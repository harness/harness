package datastore

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/drone/drone/model"
	"github.com/russross/meddler"
)

func (db *datastore) ReadLog(job *model.Job) (io.ReadCloser, error) {
	var log = new(model.Log)
	var err = meddler.QueryRow(db, log, rebind(logQuery), job.ID)
	var buf = bytes.NewBuffer(log.Data)
	return ioutil.NopCloser(buf), err
}

func (db *datastore) WriteLog(job *model.Job, r io.Reader) error {
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
