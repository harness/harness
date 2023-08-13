// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package logs

import (
	"bytes"
	"context"
	"io"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var _ store.LogStore = (*logStore)(nil)

// not used out of this package.
type logs struct {
	ID   int64  `db:"log_id"`
	Data []byte `db:"log_data"`
}

// NewLogStore returns a new LogStore.
func NewDatabaseLogStore(db *sqlx.DB) *logStore {
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
		return nil, database.ProcessSQLErrorf(err, "Failed to find log")
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
			,:log_data`
	data, err := io.ReadAll(r)
	if err != nil {
		return errors.Wrap(err, "could not read log data")
	}
	params := &logs{
		ID:   stepID,
		Data: data,
	}

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(logInsertStmt, params)
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to bind log object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&params.ID); err != nil {
		return database.ProcessSQLErrorf(err, "log query failed")
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
		return errors.Wrap(err, "could not read log data")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(logUpdateStmt, &logs{ID: stepID, Data: data})
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to bind log object")
	}

	_, err = db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to update log")
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
		return database.ProcessSQLErrorf(err, "Could not delete log")
	}

	return nil
}
