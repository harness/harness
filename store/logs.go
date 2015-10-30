package store

import (
	"io"

	"github.com/drone/drone/model"
	"golang.org/x/net/context"
)

type LogStore interface {
	// Read reads the Job logs from the datastore.
	Read(*model.Job) (io.ReadCloser, error)

	// Write writes the job logs to the datastore.
	Write(*model.Job, io.Reader) error
}

func ReadLog(c context.Context, job *model.Job) (io.ReadCloser, error) {
	return FromContext(c).Logs().Read(job)
}

func WriteLog(c context.Context, job *model.Job, r io.Reader) error {
	return FromContext(c).Logs().Write(job, r)
}
