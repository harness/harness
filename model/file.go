// Copyright 2018 Drone.IO Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	ID      int64  `json:"id"      meddler:"file_id,pk"`
	BuildID int64  `json:"-"       meddler:"file_build_id"`
	ProcID  int64  `json:"proc_id" meddler:"file_proc_id"`
	PID     int    `json:"pid"     meddler:"file_pid"`
	Name    string `json:"name"    meddler:"file_name"`
	Size    int    `json:"size"    meddler:"file_size"`
	Mime    string `json:"mime"    meddler:"file_mime"`
	Time    int64  `json:"time"    meddler:"file_time"`
	Passed  int    `json:"passed"  meddler:"file_meta_passed"`
	Failed  int    `json:"failed"  meddler:"file_meta_failed"`
	Skipped int    `json:"skipped" meddler:"file_meta_skipped"`
}
