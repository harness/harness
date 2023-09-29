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

package dbtx

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
)

// runnerDB executes individual sqlx database calls wrapped with the locker calls (Lock/Unlock)
// or a transaction wrapped with the locker calls (RLock/RUnlock or Lock/Unlock depending on the transaction type).
type runnerDB struct {
	db transactor
	mx locker
}

var _ AccessorTx = runnerDB{}

func (r runnerDB) WithTx(ctx context.Context, txFn func(context.Context) error, opts ...interface{}) error {
	var txOpts *sql.TxOptions
	for _, opt := range opts {
		if v, ok := opt.(*sql.TxOptions); ok {
			txOpts = v
		}
	}

	if txOpts == nil {
		txOpts = TxDefault
	}

	if txOpts.ReadOnly {
		r.mx.RLock()
		defer r.mx.RUnlock()
	} else {
		r.mx.Lock()
		defer r.mx.Unlock()
	}

	tx, err := r.db.startTx(ctx, txOpts)
	if err != nil {
		return err
	}

	rtx := &runnerTx{
		TransactionAccessor: tx,
		commit:              false,
		rollback:            false,
	}

	defer func() {
		if rtx.commit || rtx.rollback {
			return
		}
		_ = tx.Rollback() // ignoring the rollback error
	}()

	err = txFn(context.WithValue(ctx, ctxKeyTx{}, TransactionAccessor(rtx)))
	if err != nil {
		return err
	}

	if !rtx.commit && !rtx.rollback {
		err = rtx.Commit()
		if errors.Is(err, sql.ErrTxDone) {
			// Check if the transaction failed because of the context, if yes return the ctx error.
			if ctxErr := ctx.Err(); errors.Is(ctxErr, context.Canceled) || errors.Is(ctxErr, context.DeadlineExceeded) {
				err = ctxErr
			}
		}
	}

	return err
}

func (r runnerDB) DriverName() string {
	return r.db.DriverName()
}

func (r runnerDB) Rebind(query string) string {
	return r.db.Rebind(query)
}

func (r runnerDB) BindNamed(query string, arg interface{}) (string, []interface{}, error) {
	return r.db.BindNamed(query, arg)
}

func (r runnerDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	r.mx.Lock()
	defer r.mx.Unlock()
	return r.db.QueryContext(ctx, query, args...)
}

func (r runnerDB) QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	r.mx.Lock()
	defer r.mx.Unlock()
	return r.db.QueryxContext(ctx, query, args...)
}

func (r runnerDB) QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	r.mx.Lock()
	defer r.mx.Unlock()
	return r.db.QueryRowxContext(ctx, query, args...)
}

func (r runnerDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	r.mx.Lock()
	defer r.mx.Unlock()
	return r.db.ExecContext(ctx, query, args...)
}

func (r runnerDB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	r.mx.Lock()
	defer r.mx.Unlock()
	return r.db.QueryRowContext(ctx, query, args...)
}

func (r runnerDB) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	r.mx.Lock()
	defer r.mx.Unlock()
	return r.db.PrepareContext(ctx, query)
}

func (r runnerDB) PreparexContext(ctx context.Context, query string) (*sqlx.Stmt, error) {
	r.mx.Lock()
	defer r.mx.Unlock()
	return r.db.PreparexContext(ctx, query)
}

func (r runnerDB) PrepareNamedContext(ctx context.Context, query string) (*sqlx.NamedStmt, error) {
	r.mx.Lock()
	defer r.mx.Unlock()
	return r.db.PrepareNamedContext(ctx, query)
}

func (r runnerDB) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	r.mx.Lock()
	defer r.mx.Unlock()
	return r.db.GetContext(ctx, dest, query, args...)
}

func (r runnerDB) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	r.mx.Lock()
	defer r.mx.Unlock()
	return r.db.SelectContext(ctx, dest, query, args...)
}

// runnerTx executes sqlx database transaction calls.
// Locking is not used because runnerDB locks the entire transaction.
type runnerTx struct {
	TransactionAccessor
	commit   bool
	rollback bool
}

var _ TransactionAccessor = (*runnerTx)(nil)

func (r *runnerTx) Commit() error {
	err := r.TransactionAccessor.Commit()
	if err == nil {
		r.commit = true
	}
	return err
}

func (r *runnerTx) Rollback() error {
	err := r.TransactionAccessor.Rollback()
	if err == nil {
		r.rollback = true
	}
	return err
}
