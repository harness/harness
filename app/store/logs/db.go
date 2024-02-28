// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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
	"fmt"
	"io"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/jmoiron/sqlx"
)

var _ store.LogStore = (*logStore)(nil)

// not used out of this package.
type logs struct {
	ID   int64  `db:"log_id"`
	Data []byte `db:"log_data"`
}

// NewDatabaseLogStore returns a new LogStore.
func NewDatabaseLogStore(db *sqlx.DB) store.LogStore {
	return &logStore{
		db: db,
	}
}

type logStore struct {
	db *sqlx.DB
}

// Find returns a log given a log ID.
func (s *logStore) Find(ctx context.Context, stepID int64) (io.ReadCloser, error) {
	const findQueryStmt = `
			SELECT
			log_id, log_data
			FROM logs
			WHERE log_id = $1`
	db := dbtx.GetAccessor(ctx, s.db)

	var err error
	dst := new(logs)
	if err = db.GetContext(ctx, dst, findQueryStmt, stepID); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find log")
	}
	return io.NopCloser(
		bytes.NewBuffer(dst.Data),
	), err
}

// Create creates a log.
func (s *logStore) Create(ctx context.Context, stepID int64, r io.Reader) error {
	const logInsertStmt = `
		INSERT INTO logs (
			log_id
			,log_data
		) values (
			:log_id
			,:log_data
		)`
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("could not read log data: %w", err)
	}
	params := &logs{
		ID:   stepID,
		Data: data,
	}

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(logInsertStmt, params)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind log object")
	}

	if _, err := db.ExecContext(ctx, query, arg...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "log query failed")
	}

	return nil
}

// Update overrides existing logs data.
func (s *logStore) Update(ctx context.Context, stepID int64, r io.Reader) error {
	const logUpdateStmt = `
	UPDATE logs
	SET
		log_data = :log_data
	WHERE log_id = :log_id`
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("could not read log data: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(logUpdateStmt, &logs{ID: stepID, Data: data})
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind log object")
	}

	_, err = db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update log")
	}

	return nil
}

// Delete deletes a log given a log ID.
func (s *logStore) Delete(ctx context.Context, stepID int64) error {
	const logDeleteStmt = `
		DELETE FROM logs
		WHERE log_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, logDeleteStmt, stepID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Could not delete log")
	}

	return nil
}
