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

package database

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/harness/gitness/app/store/database/migrate"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// rdbSetupDB creates a migrated file based SQLite database in a temp dir.
func rdbSetupDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "rdb_test.db")
	db, err := sqlx.Connect("sqlite3", dsn)
	require.NoError(t, err)
	_, err = db.ExecContext(context.Background(), "PRAGMA busy_timeout = 5000;")
	require.NoError(t, err)
	require.NoError(t, migrate.Migrate(context.Background(), db))
	_, err = db.ExecContext(context.Background(), "PRAGMA foreign_keys = ON;")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}
