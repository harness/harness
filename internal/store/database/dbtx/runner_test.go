// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package dbtx

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

//nolint:gocognit
func TestWithTx(t *testing.T) {
	errTest := errors.New("dummy error")

	tests := []struct {
		name            string
		fn              func(tx Transaction) error
		errCommit       error
		cancelCtx       bool
		expectErr       error
		expectCommitted bool
		expectRollback  bool
	}{
		{
			name:            "successful",
			fn:              func(Transaction) error { return nil },
			expectCommitted: true,
		},
		{
			name:           "err-in-transaction",
			fn:             func(Transaction) error { return errTest },
			expectErr:      errTest,
			expectRollback: true,
		},
		{
			name:           "commit-failed",
			fn:             func(Transaction) error { return nil },
			errCommit:      errTest,
			expectErr:      errTest,
			expectRollback: true,
		},
		{
			name:           "commit-failed-tx-done",
			fn:             func(Transaction) error { return nil },
			errCommit:      sql.ErrTxDone,
			expectErr:      sql.ErrTxDone,
			expectRollback: true,
		},
		{
			name:           "commit-failed-ctx-cancelled",
			fn:             func(Transaction) error { return nil },
			errCommit:      sql.ErrTxDone,
			cancelCtx:      true,
			expectErr:      context.Canceled,
			expectRollback: true,
		},
		{
			name:           "panic-in-transaction",
			fn:             func(Transaction) error { panic("dummy panic") },
			expectRollback: true,
		},
		{
			name: "commit-in-transaction",
			fn: func(tx Transaction) error {
				_ = tx.Commit()
				return nil
			},
			expectCommitted: true,
		},
		{
			name: "commit-in-transaction-fn-returns-err",
			fn: func(tx Transaction) error {
				_ = tx.Commit()
				return errTest
			},
			expectErr:       errTest,
			expectCommitted: true,
		},
		{
			name: "rollback-in-transaction",
			fn: func(tx Transaction) error {
				_ = tx.Rollback()
				return nil
			},
			expectRollback: true,
		},
		{
			name: "rollback-in-transaction-fn-returns-err",
			fn: func(tx Transaction) error {
				_ = tx.Rollback()
				return errTest
			},
			expectErr:      errTest,
			expectRollback: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mock := &dbMock{
				t:         t,
				errCommit: test.errCommit,
			}
			run := &runnerDB{
				db: mock,
				mx: lockerNop{},
			}

			ctx, cancelFn := context.WithCancel(context.Background())
			defer cancelFn()

			var err error

			func() {
				defer func() {
					_ = recover()
				}()

				err = run.WithTx(ctx, func(ctx context.Context) error {
					if test.cancelCtx {
						cancelFn()
					}
					return test.fn(GetTransaction(ctx))
				})
			}()

			tx := mock.createdTx
			if tx == nil {
				t.Error("did not start a transaction")
				return
			}

			if !tx.finished {
				t.Error("transaction not finished")
			}

			if want, got := test.expectErr, err; !errors.Is(got, want) {
				t.Errorf("expected error %v, but got %v", want, got)
			}

			if want, got := test.expectCommitted, tx.committed; want != got {
				t.Errorf("expected committed %t, but got %t", want, got)
			}

			if want, got := test.expectRollback, tx.rollback; want != got {
				t.Errorf("expected rollback %t, but got %t", want, got)
			}
		})
	}
}

type dbMock struct {
	*sqlx.DB  // only to fulfill the Accessor interface, will be nil
	t         *testing.T
	errCommit error
	createdTx *txMock
}

var _ transactor = (*dbMock)(nil)

func (d *dbMock) startTx(context.Context, *sql.TxOptions) (Tx, error) {
	d.createdTx = &txMock{
		t:         d.t,
		errCommit: d.errCommit,
		finished:  false,
		committed: false,
		rollback:  false,
	}
	return d.createdTx, nil
}

type txMock struct {
	*sqlx.Tx  // only to fulfill the Accessor interface, will be nil
	t         *testing.T
	errCommit error
	finished  bool
	committed bool
	rollback  bool
}

var _ Tx = (*txMock)(nil)

func (tx *txMock) Commit() error {
	if tx.finished {
		tx.t.Error("Committing an already finished transaction")
		return nil
	}
	if tx.errCommit == nil {
		tx.finished = true
		tx.committed = true
	}
	return tx.errCommit
}

func (tx *txMock) Rollback() error {
	if tx.finished {
		tx.t.Error("Rolling back an already finished transaction")
		return nil
	}
	tx.finished = true
	tx.rollback = true
	return nil
}

func TestLocking(t *testing.T) {
	const dummyQuery = ""
	tests := []struct {
		name string
		fn   func(db Transactor, l *lockerCounter)
	}{
		{
			name: "exec-lock",
			fn: func(db Transactor, l *lockerCounter) {
				ctx := context.Background()
				_, _ = db.ExecContext(ctx, dummyQuery)
				_, _ = db.ExecContext(ctx, dummyQuery)
				_, _ = db.ExecContext(ctx, dummyQuery)

				assert.Zero(t, l.RLocks)
				assert.Zero(t, l.RUnlocks)
				assert.Equal(t, 3, l.Locks)
				assert.Equal(t, 3, l.Unlocks)
			},
		},
		{
			name: "tx-lock",
			fn: func(db Transactor, l *lockerCounter) {
				ctx := context.Background()
				_ = db.WithTx(ctx, func(ctx context.Context) error {
					_, _ = GetAccessor(ctx, nil).ExecContext(ctx, dummyQuery)
					_, _ = GetAccessor(ctx, nil).ExecContext(ctx, dummyQuery)
					return nil
				})

				assert.Zero(t, l.RLocks)
				assert.Zero(t, l.RUnlocks)
				assert.Equal(t, 1, l.Locks)
				assert.Equal(t, 1, l.Unlocks)
			},
		},
		{
			name: "tx-read-lock",
			fn: func(db Transactor, l *lockerCounter) {
				ctx := context.Background()
				_ = db.WithTx(ctx, func(ctx context.Context) error {
					_, _ = GetAccessor(ctx, nil).QueryContext(ctx, dummyQuery)
					_, _ = GetAccessor(ctx, nil).QueryContext(ctx, dummyQuery)
					return nil
				}, TxDefaultReadOnly)

				assert.Equal(t, 1, l.RLocks)
				assert.Equal(t, 1, l.RUnlocks)
				assert.Zero(t, l.Locks)
				assert.Zero(t, l.Unlocks)
			},
		},
	}

	for _, test := range tests {
		l := &lockerCounter{}
		t.Run(test.name, func(t *testing.T) {
			test.fn(runnerDB{
				db: dbMockNop{},
				mx: l,
			}, l)
		})
	}
}

type lockerCounter struct {
	Locks    int
	Unlocks  int
	RLocks   int
	RUnlocks int
}

func (l *lockerCounter) Lock()    { l.Locks++ }
func (l *lockerCounter) Unlock()  { l.Unlocks++ }
func (l *lockerCounter) RLock()   { l.RLocks++ }
func (l *lockerCounter) RUnlock() { l.RUnlocks++ }

type dbMockNop struct{}

func (dbMockNop) DriverName() string                                           { return "" }
func (dbMockNop) Rebind(string) string                                         { return "" }
func (dbMockNop) BindNamed(string, interface{}) (string, []interface{}, error) { return "", nil, nil }

//nolint:nilnil // it's a mock
func (dbMockNop) QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error) {
	return nil, nil
}

//nolint:nilnil // it's a mock
func (dbMockNop) QueryxContext(context.Context, string, ...interface{}) (*sqlx.Rows, error) {
	return nil, nil
}
func (dbMockNop) QueryRowxContext(context.Context, string, ...interface{}) *sqlx.Row { return nil }
func (dbMockNop) ExecContext(context.Context, string, ...interface{}) (sql.Result, error) {
	return nil, nil
}
func (dbMockNop) QueryRowContext(context.Context, string, ...any) *sql.Row {
	return nil
}

//nolint:nilnil // it's a mock
func (dbMockNop) PrepareContext(context.Context, string) (*sql.Stmt, error) {
	return nil, nil
}

//nolint:nilnil // it's a mock
func (dbMockNop) PreparexContext(context.Context, string) (*sqlx.Stmt, error) {
	return nil, nil
}

//nolint:nilnil // it's a mock
func (dbMockNop) PrepareNamedContext(context.Context, string) (*sqlx.NamedStmt, error) {
	return nil, nil
}
func (dbMockNop) GetContext(context.Context, interface{}, string, ...interface{}) error {
	return nil
}
func (dbMockNop) SelectContext(context.Context, interface{}, string, ...interface{}) error {
	return nil
}

func (dbMockNop) Commit() error   { return nil }
func (dbMockNop) Rollback() error { return nil }

func (d dbMockNop) startTx(context.Context, *sql.TxOptions) (Tx, error) { return d, nil }
