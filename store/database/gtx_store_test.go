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

package database

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/harness/gitness/store"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3" // sqlite driver for the tests.
)

func gtxNewSQLite(ctx context.Context, t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := Connect(ctx, "sqlite3", "file:"+filepath.Join(t.TempDir(), "gtx.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	return db
}

func TestGtxPrepareDatasourceForDriver(t *testing.T) {
	tests := []struct {
		name       string
		driver     string
		datasource string
		want       string
		wantErr    bool
	}{
		{
			name: "sqlite3 gets foreign keys enabled", driver: "sqlite3", datasource: "file:test.db",
			want: "file:test.db?_foreign_keys=on",
		},
		{
			name: "sqlite3 keeps other settings", driver: "sqlite3", datasource: "file:test.db?cache=shared",
			want: "file:test.db?_foreign_keys=on&cache=shared",
		},
		{
			name: "sqlite3 overwrites foreign keys setting", driver: "sqlite3", datasource: "file:test.db?_foreign_keys=off",
			want: "file:test.db?_foreign_keys=on",
		},
		{name: "sqlite3 invalid url", driver: "sqlite3", datasource: "://bad", wantErr: true},
		{name: "postgres is untouched", driver: "postgres", datasource: "host=a user=b", want: "host=a user=b"},
		{name: "unknown driver is untouched", driver: "gtx", datasource: "anything", want: "anything"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := prepareDatasourceForDriver(test.driver, test.datasource)
			if test.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestGtxConnect(t *testing.T) {
	ctx := context.Background()

	t.Run("sqlite", func(t *testing.T) {
		db := gtxNewSQLite(ctx, t)

		var one int
		require.NoError(t, db.GetContext(ctx, &one, "SELECT 1"))
		assert.Equal(t, 1, one)

		var fk int
		require.NoError(t, db.GetContext(ctx, &fk, "PRAGMA foreign_keys"))
		assert.Equal(t, 1, fk, "foreign keys must be enabled")
		assert.Equal(t, "sqlite3", db.DriverName())
	})

	t.Run("invalid datasource", func(t *testing.T) {
		_, err := Connect(ctx, "sqlite3", "://bad")
		require.ErrorContains(t, err, "failed to prepare datasource")
	})

	t.Run("unknown driver", func(t *testing.T) {
		_, err := Connect(ctx, "gtx-unknown-driver", "x")
		require.ErrorContains(t, err, "failed to open the db")
	})

	t.Run("canceled context", func(t *testing.T) {
		cctx, cancel := context.WithCancel(ctx)
		cancel()

		_, err := Connect(cctx, "sqlite3", "file:"+filepath.Join(t.TempDir(), "gtx.db"))
		require.ErrorIs(t, err, context.Canceled)
		require.ErrorContains(t, err, "failed to ping the db")
	})
}

func TestGtxConnectAndMigrate(t *testing.T) {
	ctx := context.Background()

	t.Run("migrators run in order", func(t *testing.T) {
		var calls []string
		create := func(ctx context.Context, db *sqlx.DB) error {
			calls = append(calls, "create")
			_, err := db.ExecContext(ctx, "CREATE TABLE gtx (id INTEGER PRIMARY KEY, name TEXT UNIQUE)")
			return err
		}
		insert := func(ctx context.Context, db *sqlx.DB) error {
			calls = append(calls, "insert")
			_, err := db.ExecContext(ctx, "INSERT INTO gtx (name) VALUES ('a')")
			return err
		}

		db, err := ConnectAndMigrate(ctx, "sqlite3", "file:"+filepath.Join(t.TempDir(), "gtx.db"), create, insert)
		require.NoError(t, err)
		t.Cleanup(func() { _ = db.Close() })

		assert.Equal(t, []string{"create", "insert"}, calls)

		var n int
		require.NoError(t, db.GetContext(ctx, &n, "SELECT COUNT(*) FROM gtx"))
		assert.Equal(t, 1, n)
	})

	t.Run("migrator failure", func(t *testing.T) {
		migErr := errors.New("gtx migration failure")
		db, err := ConnectAndMigrate(ctx, "sqlite3", "file:"+filepath.Join(t.TempDir(), "gtx.db"),
			func(context.Context, *sqlx.DB) error { return migErr })
		require.ErrorIs(t, err, migErr)
		require.ErrorContains(t, err, "failed to setup the db")
		assert.Nil(t, db)
	})

	t.Run("connection failure", func(t *testing.T) {
		db, err := ConnectAndMigrate(ctx, "gtx-unknown-driver", "x")
		require.Error(t, err)
		assert.Nil(t, db)
	})
}

func TestGtxMust(t *testing.T) {
	db := gtxNewSQLite(context.Background(), t)
	assert.Same(t, db, Must(db, nil))

	cause := errors.New("gtx connect failure")
	assert.PanicsWithError(t, "gtx connect failure", func() { Must(nil, cause) })
}

func TestGtxProcessSQLErrorf_RealDatabase(t *testing.T) {
	ctx := context.Background()
	db := gtxNewSQLite(ctx, t)

	for _, stmt := range []string{
		"CREATE TABLE gtx_parent (id INTEGER PRIMARY KEY)",
		"CREATE TABLE gtx_child (id INTEGER PRIMARY KEY, parent_id INTEGER NOT NULL REFERENCES gtx_parent(id), " +
			"name TEXT UNIQUE)",
		"INSERT INTO gtx_parent (id) VALUES (1)",
		"INSERT INTO gtx_child (id, parent_id, name) VALUES (1, 1, 'a')",
	} {
		_, err := db.ExecContext(ctx, stmt)
		require.NoError(t, err, stmt)
	}

	tests := []struct {
		name  string
		query func() error
		want  error
	}{
		{
			name: "unique violation",
			query: func() error {
				_, err := db.ExecContext(ctx, "INSERT INTO gtx_child (id, parent_id, name) VALUES (2, 1, 'a')")
				return err
			},
			want: store.ErrDuplicate,
		},
		{
			name: "primary key violation",
			query: func() error {
				_, err := db.ExecContext(ctx, "INSERT INTO gtx_child (id, parent_id, name) VALUES (1, 1, 'b')")
				return err
			},
			want: store.ErrDuplicate,
		},
		{
			name: "foreign key violation",
			query: func() error {
				_, err := db.ExecContext(ctx, "INSERT INTO gtx_child (id, parent_id, name) VALUES (3, 99, 'c')")
				return err
			},
			want: store.ErrForeignKeyViolation,
		},
		{
			name: "no rows",
			query: func() error {
				var id int
				return db.GetContext(ctx, &id, "SELECT id FROM gtx_child WHERE id = 12345")
			},
			want: store.ErrResourceNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.query()
			require.Error(t, err)

			got := ProcessSQLErrorf(ctx, err, "failed to %s", "query")
			require.ErrorIs(t, got, test.want)
			assert.Contains(t, got.Error(), "failed to query: ")
		})
	}

	t.Run("other errors are wrapped as is", func(t *testing.T) {
		_, err := db.ExecContext(ctx, "INSERT INTO gtx_child (id, parent_id) VALUES (4, NULL)")
		require.Error(t, err)

		got := ProcessSQLErrorf(ctx, err, "failed")
		require.Error(t, got)
		assert.False(t, errors.Is(got, store.ErrDuplicate))
		assert.False(t, errors.Is(got, store.ErrForeignKeyViolation))
		assert.False(t, errors.Is(got, store.ErrResourceNotFound))
		require.ErrorIs(t, got, err)
	})

	t.Run("format arguments", func(t *testing.T) {
		got := ProcessSQLErrorf(ctx, errors.New("boom"), "op %d on %q", 5, "x")
		assert.EqualError(t, got, fmt.Sprintf("op 5 on %q: boom", "x"))
	})
}

func TestGtxBuilder(t *testing.T) {
	sql, args, err := Builder.Select("id").From("t").Where("a = ? AND b = ?", 1, "x").ToSql()
	require.NoError(t, err)
	assert.Equal(t, "SELECT id FROM t WHERE a = $1 AND b = $2", sql)
	assert.Equal(t, []any{1, "x"}, args)
	assert.Equal(t, "FOR UPDATE", SQLForUpdate)
}
