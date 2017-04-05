package datastore

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/drone/drone/model"
	"github.com/russross/meddler"
)

func (db *datastore) LogFind(proc *model.Proc) (io.ReadCloser, error) {
	var log = new(logData)
	var err = meddler.QueryRow(db, log, rebind(logQuery), proc.ID)
	var buf = bytes.NewBuffer(log.Data)
	return ioutil.NopCloser(buf), err
}

func (db *datastore) LogSave(proc *model.Proc, r io.Reader) error {
	var log = new(logData)
	var err = meddler.QueryRow(db, log, rebind(logQuery), proc.ID)
	if err != nil {
		log = &logData{ProcID: proc.ID}
	}
	log.Data, _ = ioutil.ReadAll(r)
	return meddler.Save(db, logTable, log)
}

type logData struct {
	ID     int64  `meddler:"log_id,pk"`
	ProcID int64  `meddler:"log_job_id"`
	Data   []byte `meddler:"log_data"`
}

const logTable = "logs"

const logQuery = `
SELECT *
FROM logs
WHERE log_job_id=?
LIMIT 1
`
