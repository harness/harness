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

package migrations

import (
	"errors"
	"io"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	dbstub "github.com/golang-migrate/migrate/v4/database/stub"
	"github.com/golang-migrate/migrate/v4/source"
	srcstub "github.com/golang-migrate/migrate/v4/source/stub"
	"github.com/stretchr/testify/require"
)

// rdbFailingDriver wraps a database driver and fails every migration run.
type rdbFailingDriver struct {
	database.Driver
}

func (rdbFailingDriver) Run(_ io.Reader) error {
	return errors.New("boom")
}

func rdbNewSource(t *testing.T, versions ...uint) source.Driver {
	t.Helper()
	src, err := (&srcstub.Stub{}).Open("stub://")
	require.NoError(t, err)
	stub, ok := src.(*srcstub.Stub)
	require.True(t, ok)
	for _, v := range versions {
		require.True(t, stub.Migrations.Append(&source.Migration{
			Version: v, Direction: source.Up, Identifier: "up",
		}))
		require.True(t, stub.Migrations.Append(&source.Migration{
			Version: v, Direction: source.Down, Identifier: "down",
		}))
	}
	return src
}

func rdbNewDB(t *testing.T) *dbstub.Stub {
	t.Helper()
	drv, err := (&dbstub.Stub{}).Open("stub://")
	require.NoError(t, err)
	stub, ok := drv.(*dbstub.Stub)
	require.True(t, ok)
	return stub
}

func TestRdbMigrate_AppliesAndIsIdempotent(t *testing.T) {
	db := rdbNewDB(t)
	m, err := migrate.NewWithInstance("stub", rdbNewSource(t, 1, 2), "stub", db)
	require.NoError(t, err)

	require.NoError(t, Migrate(m))
	version, dirty, err := db.Version()
	require.NoError(t, err)
	require.Equal(t, 2, version)
	require.False(t, dirty)
	require.Len(t, db.MigrationSequence, 2)

	// A second run has nothing to do and is not an error.
	require.NoError(t, Migrate(m))
	require.Len(t, db.MigrationSequence, 2)
}

func TestRdbMigrate_RecoversDirtyVersion(t *testing.T) {
	db := rdbNewDB(t)
	m, err := migrate.NewWithInstance("stub", rdbNewSource(t, 1, 2), "stub", db)
	require.NoError(t, err)
	require.NoError(t, db.SetVersion(2, true))

	require.NoError(t, Migrate(m))
	version, dirty, err := db.Version()
	require.NoError(t, err)
	require.Equal(t, 2, version)
	require.False(t, dirty)
	// The dirty version was forced back one step and migration 2 re-applied.
	require.Len(t, db.MigrationSequence, 1)
}

func TestRdbMigrate_MissingMigrationFileIsSkipped(t *testing.T) {
	db := rdbNewDB(t)
	m, err := migrate.NewWithInstance("stub", rdbNewSource(t), "stub", db)
	require.NoError(t, err)
	require.NoError(t, db.SetVersion(5, false))

	require.NoError(t, Migrate(m))
}

func TestRdbMigrate_ReturnsRunFailure(t *testing.T) {
	db := rdbNewDB(t)
	m, err := migrate.NewWithInstance("stub", rdbNewSource(t, 1), "stub", rdbFailingDriver{Driver: db})
	require.NoError(t, err)

	err = Migrate(m)
	require.Error(t, err)
	require.ErrorContains(t, err, "error when migration up")
	require.ErrorContains(t, err, "boom")
}
