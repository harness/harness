package datastore

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/drone/drone/model"
	"github.com/drone/drone/store/datastore/sql"
	"github.com/russross/meddler"
)

func (db *datastore) LogFind(proc *model.Proc) (io.ReadCloser, error) {
	stmt := sql.Lookup(db.driver, "logs-find-proc")
	data := new(logData)
	err := meddler.QueryRow(db, data, stmt, proc.ID)
	buf := bytes.NewBuffer(data.Data)
	return ioutil.NopCloser(buf), err
}

func (db *datastore) LogSave(proc *model.Proc, r io.Reader) error {
	stmt := sql.Lookup(db.driver, "logs-find-proc")
	data := new(logData)
	err := meddler.QueryRow(db, data, stmt, proc.ID)
	if err != nil {
		data = &logData{ProcID: proc.ID}
	}
	data.Data, _ = ioutil.ReadAll(r)
	return meddler.Save(db, "logs", data)
}

type logData struct {
	ID     int64  `meddler:"log_id,pk"`
	ProcID int64  `meddler:"log_job_id"`
	Data   []byte `meddler:"log_data"`
}
