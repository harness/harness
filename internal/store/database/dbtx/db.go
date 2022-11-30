// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package dbtx

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

// New returns new database Runner interface.
func New(db *sqlx.DB) Transactor {
	run := &runnerDB{sqlDB{db}}
	return run
}

// transactor is combines data access capabilities with transaction starting.
type transactor interface {
	Accessor
	startTx(ctx context.Context, opts *sql.TxOptions) (Tx, error)
}

// sqlDB is a wrapper for the sqlx.DB that implements the transactor interface.
type sqlDB struct {
	*sqlx.DB
}

var _ transactor = (*sqlDB)(nil)

func (db sqlDB) startTx(ctx context.Context, opts *sql.TxOptions) (Tx, error) {
	tx, err := db.DB.BeginTxx(ctx, opts)
	return tx, err
}
