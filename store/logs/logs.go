// Copyright 2019 Drone IO, Inc.
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

package logs

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"

	"github.com/drone/drone/core"
	"github.com/drone/drone/store/shared/db"
)

// New returns a new LogStore.
func New(db *db.DB) core.LogStore {
	return &logStore{db}
}

type logStore struct {
	db *db.DB
}

func (s *logStore) Find(ctx context.Context, step int64) (io.ReadCloser, error) {
	out := &logs{ID: step}
	err := s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		query, args, err := binder.BindNamed(queryKey, out)
		if err != nil {
			return err
		}
		row := queryer.QueryRow(query, args...)
		return scanRow(row, out)
	})
	return ioutil.NopCloser(
		bytes.NewBuffer(out.Data),
	), err
}

func (s *logStore) Create(ctx context.Context, step int64, r io.Reader) error {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	return s.db.Lock(func(execer db.Execer, binder db.Binder) error {
		params := &logs{
			ID:   step,
			Data: data,
		}
		stmt, args, err := binder.BindNamed(stmtInsert, params)
		if err != nil {
			return err
		}
		_, err = execer.Exec(stmt, args...)
		return err
	})
}

func (s *logStore) Update(ctx context.Context, step int64, r io.Reader) error {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	return s.db.Lock(func(execer db.Execer, binder db.Binder) error {
		params := &logs{
			ID:   step,
			Data: data,
		}
		stmt, args, err := binder.BindNamed(stmtUpdate, params)
		if err != nil {
			return err
		}
		_, err = execer.Exec(stmt, args...)
		return err
	})
}

func (s *logStore) Delete(ctx context.Context, step int64) error {
	return s.db.Lock(func(execer db.Execer, binder db.Binder) error {
		params := &logs{
			ID: step,
		}
		stmt, args, err := binder.BindNamed(stmtDelete, params)
		if err != nil {
			return err
		}
		_, err = execer.Exec(stmt, args...)
		return err
	})
}

type logs struct {
	ID   int64  `db:"log_id"`
	Data []byte `db:"log_data"`
}

const queryKey = `
SELECT
 log_id
,log_data
FROM logs
WHERE log_id = :log_id
`

const stmtInsert = `
INSERT INTO logs (
 log_id
,log_data
) VALUES (
 :log_id
,:log_data
)
`

const stmtUpdate = `
UPDATE logs
SET log_data = :log_data
WHERE log_id = :log_id
`

const stmtDelete = `
DELETE FROM logs
WHERE log_id = :log_id
`
