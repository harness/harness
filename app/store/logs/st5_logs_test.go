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

package logs_test

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/harness/gitness/app/store/database/migrate"
	"github.com/harness/gitness/app/store/logs"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3" // sqlite driver
)

var errSt5Read = errors.New("st5 read failed")

type st5FailingReader struct{}

func (st5FailingReader) Read([]byte) (int, error) { return 0, errSt5Read }

func TestSt5DatabaseLogStoreErrors(t *testing.T) {
	ctx := context.Background()

	db, err := sqlx.Connect("sqlite3", filepath.Join(t.TempDir(), "st5logs.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, migrate.Migrate(ctx, db))

	s := logs.NewDatabaseLogStore(db)

	t.Run("unreadable data", func(t *testing.T) {
		require.ErrorIs(t, s.Create(ctx, 1, st5FailingReader{}), errSt5Read)
		require.ErrorIs(t, s.Update(ctx, 1, st5FailingReader{}), errSt5Read)
	})

	t.Run("closed database", func(t *testing.T) {
		require.NoError(t, db.Close())

		require.Error(t, s.Create(ctx, 1, bytes.NewBufferString("log")))
		require.Error(t, s.Update(ctx, 1, bytes.NewBufferString("log")))
		require.Error(t, s.Delete(ctx, 1))
		_, err := s.Find(ctx, 1)
		require.Error(t, err)
	})
}
