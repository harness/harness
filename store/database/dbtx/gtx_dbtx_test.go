// Copyright 2026 Harness, Inc.
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

//go:build !nosqlite
// +build !nosqlite

package dbtx

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3" // sqlite driver for the tests.
)

type gtxRow struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
}

func gtxNewDB(ctx context.Context, t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.ConnectContext(ctx, "sqlite3", "file:"+filepath.Join(t.TempDir(), "gtx.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.ExecContext(ctx, "CREATE TABLE gtx (id INTEGER PRIMARY KEY, name TEXT)")
	require.NoError(t, err)

	return db
}

func gtxCount(ctx context.Context, t *testing.T, a Accessor) int {
	t.Helper()

	var n int
	require.NoError(t, a.GetContext(ctx, &n, "SELECT COUNT(*) FROM gtx"))

	return n
}

func TestGtxAccessor_Queries(t *testing.T) {
	ctx := context.Background()
	a := New(gtxNewDB(ctx, t))

	assert.Equal(t, "sqlite3", a.DriverName())
	assert.Equal(t, "SELECT ?", a.Rebind("SELECT ?"), "sqlite uses question mark placeholders")

	query, args, err := a.BindNamed("SELECT :id", map[string]any{"id": 5})
	require.NoError(t, err)
	assert.Equal(t, "SELECT ?", query)
	assert.Equal(t, []any{5}, args)

	res, err := a.ExecContext(ctx, "INSERT INTO gtx (id, name) VALUES (1, 'one')")
	require.NoError(t, err)
	affected, err := res.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected)

	_, err = a.NamedExecContext(ctx, "INSERT INTO gtx (id, name) VALUES (:id, :name)", gtxRow{ID: 2, Name: "two"})
	require.NoError(t, err)

	stmt, err := a.PrepareContext(ctx, "INSERT INTO gtx (id, name) VALUES (?, ?)")
	require.NoError(t, err)
	_, err = stmt.ExecContext(ctx, 3, "three")
	require.NoError(t, err)
	defer func() { _ = stmt.Close() }()

	xstmt, err := a.PreparexContext(ctx, "INSERT INTO gtx (id, name) VALUES (?, ?)")
	require.NoError(t, err)
	_, err = xstmt.ExecContext(ctx, 4, "four")
	require.NoError(t, err)
	defer func() { _ = xstmt.Close() }()

	named, err := a.PrepareNamedContext(ctx, "INSERT INTO gtx (id, name) VALUES (:id, :name)")
	require.NoError(t, err)
	_, err = named.ExecContext(ctx, gtxRow{ID: 5, Name: "five"})
	require.NoError(t, err)
	defer func() { _ = named.Close() }()

	assert.Equal(t, 5, gtxCount(ctx, t, a))

	t.Run("select", func(t *testing.T) {
		var rows []gtxRow
		require.NoError(t, a.SelectContext(ctx, &rows, "SELECT id, name FROM gtx WHERE id <= ? ORDER BY id", 2))
		assert.Equal(t, []gtxRow{{ID: 1, Name: "one"}, {ID: 2, Name: "two"}}, rows)
	})

	t.Run("get", func(t *testing.T) {
		var row gtxRow
		require.NoError(t, a.GetContext(ctx, &row, "SELECT id, name FROM gtx WHERE id = 3"))
		assert.Equal(t, gtxRow{ID: 3, Name: "three"}, row)

		err := a.GetContext(ctx, &row, "SELECT id, name FROM gtx WHERE id = 99")
		require.ErrorIs(t, err, sql.ErrNoRows)
	})

	t.Run("query", func(t *testing.T) {
		rows, err := a.QueryContext(ctx, "SELECT name FROM gtx WHERE id = 4")
		require.NoError(t, err)
		require.True(t, rows.Next())
		var name string
		require.NoError(t, rows.Scan(&name))
		assert.Equal(t, "four", name)
		require.NoError(t, rows.Err())
		defer func() { _ = rows.Close() }()
	})

	t.Run("queryx", func(t *testing.T) {
		rows, err := a.QueryxContext(ctx, "SELECT id, name FROM gtx WHERE id = 5")
		require.NoError(t, err)
		require.True(t, rows.Next())
		var row gtxRow
		require.NoError(t, rows.StructScan(&row))
		assert.Equal(t, gtxRow{ID: 5, Name: "five"}, row)
		require.NoError(t, rows.Err())
		defer func() { _ = rows.Close() }()
	})

	t.Run("query row", func(t *testing.T) {
		var name string
		require.NoError(t, a.QueryRowContext(ctx, "SELECT name FROM gtx WHERE id = 1").Scan(&name))
		assert.Equal(t, "one", name)

		var row gtxRow
		require.NoError(t, a.QueryRowxContext(ctx, "SELECT id, name FROM gtx WHERE id = 2").StructScan(&row))
		assert.Equal(t, gtxRow{ID: 2, Name: "two"}, row)
	})

	t.Run("invalid statements", func(t *testing.T) {
		_, err := a.ExecContext(ctx, "INSERT INTO gtx_missing VALUES (1)")
		require.Error(t, err)

		var n int
		require.Error(t, a.SelectContext(ctx, &n, "SELEC broken"))
	})
}

func TestGtxAccessorTx_WithTx(t *testing.T) {
	ctx := context.Background()
	a := New(gtxNewDB(ctx, t))

	t.Run("commit when the function succeeds", func(t *testing.T) {
		err := a.WithTx(ctx, func(txCtx context.Context) error {
			tx := GetAccessor(txCtx, nil)
			_, err := tx.ExecContext(txCtx, "INSERT INTO gtx (id, name) VALUES (1, 'committed')")

			return err
		})
		require.NoError(t, err)
		assert.Equal(t, 1, gtxCount(ctx, t, a))
	})

	t.Run("rollback when the function fails", func(t *testing.T) {
		txErr := errors.New("gtx tx failure")
		err := a.WithTx(ctx, func(txCtx context.Context) error {
			_, err := GetAccessor(txCtx, nil).ExecContext(txCtx, "INSERT INTO gtx (id, name) VALUES (2, 'rolled back')")
			require.NoError(t, err)

			return txErr
		})
		require.ErrorIs(t, err, txErr)
		assert.Equal(t, 1, gtxCount(ctx, t, a))
	})

	t.Run("explicit rollback", func(t *testing.T) {
		err := a.WithTx(ctx, func(txCtx context.Context) error {
			_, err := GetAccessor(txCtx, nil).ExecContext(txCtx, "INSERT INTO gtx (id, name) VALUES (3, 'explicit')")
			require.NoError(t, err)

			tx := GetTransaction(txCtx)
			require.NotNil(t, tx)

			return tx.Rollback()
		})
		require.NoError(t, err)
		assert.Equal(t, 1, gtxCount(ctx, t, a))
	})

	t.Run("explicit commit", func(t *testing.T) {
		err := a.WithTx(ctx, func(txCtx context.Context) error {
			_, err := GetAccessor(txCtx, nil).ExecContext(txCtx, "INSERT INTO gtx (id, name) VALUES (4, 'explicit')")
			require.NoError(t, err)

			return GetTransaction(txCtx).Commit()
		})
		require.NoError(t, err)
		assert.Equal(t, 2, gtxCount(ctx, t, a))
	})

	t.Run("panic rolls back", func(t *testing.T) {
		assert.Panics(t, func() {
			_ = a.WithTx(ctx, func(txCtx context.Context) error {
				_, err := GetAccessor(txCtx, nil).ExecContext(txCtx, "INSERT INTO gtx (id, name) VALUES (5, 'panic')")
				require.NoError(t, err)
				panic("gtx panic")
			})
		})
		assert.Equal(t, 2, gtxCount(ctx, t, a))
	})

	t.Run("read only transaction", func(t *testing.T) {
		err := a.WithTx(ctx, func(txCtx context.Context) error {
			assert.Equal(t, 2, gtxCount(txCtx, t, GetAccessor(txCtx, nil)))

			return nil
		}, TxDefaultReadOnly)
		require.NoError(t, err)
	})

	t.Run("unrelated options are ignored", func(t *testing.T) {
		err := a.WithTx(ctx, func(context.Context) error { return nil }, "ignored", 42)
		require.NoError(t, err)
	})

	t.Run("canceled context", func(t *testing.T) {
		cctx, cancel := context.WithCancel(ctx)
		cancel()

		err := a.WithTx(cctx, func(context.Context) error { return nil })
		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("context canceled inside the transaction", func(t *testing.T) {
		cctx, cancel := context.WithCancel(ctx)
		defer cancel()

		err := a.WithTx(cctx, func(context.Context) error {
			cancel()
			return nil
		})
		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("nested transactions see the outer accessor", func(t *testing.T) {
		err := a.WithTx(ctx, func(txCtx context.Context) error {
			acc := GetAccessor(txCtx, nil)
			_, ok := acc.(TransactionAccessor)
			assert.True(t, ok, "the accessor of a transaction can commit and roll back")

			return nil
		})
		require.NoError(t, err)
	})
}

func TestGtxContextHelpers(t *testing.T) {
	ctx := context.Background()
	db := gtxNewDB(ctx, t)

	assert.Nil(t, GetTransaction(ctx))

	acc := GetAccessor(ctx, db)
	require.NotNil(t, acc)
	assert.Equal(t, "sqlite3", acc.DriverName())
}

func TestGtxWire(t *testing.T) {
	ctx := context.Background()
	db := gtxNewDB(ctx, t)

	a := ProvideAccessorTx(db)
	require.NotNil(t, a)
	assert.NotNil(t, ProvideAccessor(a))
	assert.NotNil(t, ProvideTransactor(a))
}

func TestGtxGetLocker(t *testing.T) {
	ctx := context.Background()

	assert.True(t, needsLocking("sqlite3"))
	assert.True(t, needsLocking(""))
	assert.False(t, needsLocking("postgres"))

	assert.Same(t, &globalMx, getLocker(gtxNewDB(ctx, t)))

	pg := sqlx.NewDb(&sql.DB{}, "postgres")
	assert.Equal(t, lockerNop{}, getLocker(pg))

	nop := lockerNop{}
	assert.NotPanics(t, func() {
		for _, fn := range []func(){nop.Lock, nop.Unlock, nop.RLock, nop.RUnlock} {
			fn()
		}
	})
}

func TestGtxTxOptions(t *testing.T) {
	assert.False(t, TxDefault.ReadOnly)
	assert.True(t, TxDefaultReadOnly.ReadOnly)
	assert.Equal(t, sql.LevelRepeatableRead, TxRepeatableRead.Isolation)
	assert.Equal(t, sql.LevelSerializable, TxSerializable.Isolation)
}
