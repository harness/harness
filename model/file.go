package model

import "io"

// FileStore persists pipeline artifacts to storage.
type FileStore interface {
	FileList(*Build) ([]*File, error)
	FileFind(*Proc, string) (*File, error)
	FileRead(*Proc, string) (io.ReadCloser, error)
	FileCreate(*File, io.Reader) error
}

// File represents a pipeline artifact.
type File struct {
	ID      int64  `json:"id"       meddler:"file_id,pk"`
	BuildID int64  `json:"build_id" meddler:"file_build_id"`
	ProcID  int64  `json:"proc_id"  meddler:"file_proc_id"`
	Name    string `json:"name"     meddler:"file_name"`
	Size    int    `json:"size"     meddler:"file_size"`
	Mime    string `json:"mime"     meddler:"file_mime"`
	Time    int64  `json:"time"     meddler:"file_time"`
	// Data    []byte `json:"data"     meddler:"file_data"`
}
