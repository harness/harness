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

package system

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/app/store/database/migrate"
	"github.com/harness/gitness/types"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// cxrBase is a migrated SQLite database with a principal store and an admin user.
type cxrBase struct {
	db         *sqlx.DB
	principals *database.PrincipalStore
	admin      *types.User
}

func cxrNewBase(ctx context.Context, t *testing.T) *cxrBase {
	t.Helper()

	db, err := sqlx.Connect("sqlite3", filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.ExecContext(ctx, `PRAGMA foreign_keys = ON;`)
	require.NoError(t, err)
	_, _ = db.ExecContext(ctx, "PRAGMA busy_timeout = 5000;")
	require.NoError(t, migrate.Migrate(ctx, db))

	principals := database.NewPrincipalStore(db, store.ToLowerPrincipalUIDTransformation)
	admin := &types.User{UID: "admin", Email: "admin@example.com", DisplayName: "admin", Salt: "salt", Admin: true}
	require.NoError(t, principals.CreateUser(ctx, admin))

	return &cxrBase{db: db, principals: principals, admin: admin}
}
