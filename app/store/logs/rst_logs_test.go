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

package logs_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/harness/gitness/app/store/database/migrate"
	"github.com/harness/gitness/app/store/logs"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3" // sqlite driver
)

func rstNewDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Connect("sqlite3", filepath.Join(t.TempDir(), "logs.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	// A single connection keeps the pragma in effect. Logs reference steps through a foreign key, which is
	// not what these tests are about, so it is switched off instead of building a whole pipeline per log.
	db.SetMaxOpenConns(1)
	require.NoError(t, migrate.Migrate(context.Background(), db))

	_, err = db.Exec(`PRAGMA foreign_keys = OFF;`)
	require.NoError(t, err)

	return db
}

func rstReadAll(t *testing.T, rc io.ReadCloser) string {
	t.Helper()

	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.NoError(t, rc.Close())

	return string(data)
}

type rstFailingReader struct{}

func (rstFailingReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

func TestRstDatabaseLogStore(t *testing.T) {
	ctx := context.Background()
	logStore := logs.NewDatabaseLogStore(rstNewDB(t))

	_, err := logStore.Find(ctx, 1)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

	require.NoError(t, logStore.Create(ctx, 1, strings.NewReader("first line\n")))
	require.ErrorIs(t, logStore.Create(ctx, 1, strings.NewReader("other")), gitness_store.ErrDuplicate)

	rc, err := logStore.Find(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, "first line\n", rstReadAll(t, rc))

	require.NoError(t, logStore.Update(ctx, 1, strings.NewReader("replaced")))
	rc, err = logStore.Find(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, "replaced", rstReadAll(t, rc))

	// updating a log that does not exist is not an error and does not create it.
	require.NoError(t, logStore.Update(ctx, 2, strings.NewReader("ghost")))
	_, err = logStore.Find(ctx, 2)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

	require.Error(t, logStore.Create(ctx, 3, rstFailingReader{}))
	require.Error(t, logStore.Update(ctx, 1, rstFailingReader{}))

	require.NoError(t, logStore.Delete(ctx, 1))
	_, err = logStore.Find(ctx, 1)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

	// deleting a missing log is not an error.
	require.NoError(t, logStore.Delete(ctx, 1))
}

// rstMemLogStore is an in-memory store.LogStore which can be told to fail.
type rstMemLogStore struct {
	mu      sync.Mutex
	data    map[int64]string
	failing bool
}

var errRstStore = errors.New("store unavailable")

func newRstMemLogStore() *rstMemLogStore { return &rstMemLogStore{data: map[int64]string{}} }

func (s *rstMemLogStore) Find(_ context.Context, id int64) (io.ReadCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, ok := s.data[id]
	if s.failing || !ok {
		return nil, errRstStore
	}

	return io.NopCloser(strings.NewReader(data)), nil
}

func (s *rstMemLogStore) Create(ctx context.Context, id int64, r io.Reader) error {
	return s.Update(ctx, id, r)
}

func (s *rstMemLogStore) Update(_ context.Context, id int64, r io.Reader) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failing {
		return errRstStore
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	s.data[id] = string(data)

	return nil
}

func (s *rstMemLogStore) Delete(_ context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failing {
		return errRstStore
	}
	if _, ok := s.data[id]; !ok {
		return errRstStore
	}
	delete(s.data, id)

	return nil
}

func TestRstCombinedLogStore(t *testing.T) {
	ctx := context.Background()
	primary, secondary := newRstMemLogStore(), newRstMemLogStore()
	combined := logs.NewCombined(primary, secondary)

	// writes only go to the primary store.
	require.NoError(t, combined.Create(ctx, 1, strings.NewReader("new log")))
	require.NoError(t, combined.Update(ctx, 1, strings.NewReader("newer log")))
	require.Equal(t, "newer log", primary.data[1])
	require.Empty(t, secondary.data)

	rc, err := combined.Find(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, "newer log", rstReadAll(t, rc))

	// reads fall back to the secondary store.
	secondary.data[2] = "legacy log"
	rc, err = combined.Find(ctx, 2)
	require.NoError(t, err)
	require.Equal(t, "legacy log", rstReadAll(t, rc))

	_, err = combined.Find(ctx, 3)
	require.ErrorIs(t, err, errRstStore)

	// deleting falls back to the secondary store when the primary cannot delete.
	require.NoError(t, combined.Delete(ctx, 2))
	require.NotContains(t, secondary.data, int64(2))
	require.Error(t, combined.Delete(ctx, 2))

	require.NoError(t, combined.Delete(ctx, 1))
	require.Empty(t, primary.data)
}

// rstFakeS3 is a minimal path-style S3 endpoint that keeps objects in memory.
func rstFakeS3(t *testing.T) (*httptest.Server, map[string][]byte) {
	t.Helper()

	var mu sync.Mutex
	objects := map[string][]byte{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch r.Method {
		case http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			objects[r.URL.Path] = body
			w.Header().Set("ETag", `"etag"`)
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			body, ok := objects[r.URL.Path]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`<Error><Code>NoSuchKey</Code></Error>`))
				return
			}
			_, _ = w.Write(body)
		case http.MethodDelete:
			delete(objects, r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	t.Cleanup(srv.Close)

	return srv, objects
}

func TestRstS3LogStore(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	t.Setenv("AWS_REGION", "us-east-1")

	ctx := context.Background()
	srv, objects := rstFakeS3(t)
	s3 := logs.NewS3LogStore("bucket", "logs/prefix", srv.URL, true)

	_, err := s3.Find(ctx, 5)
	require.Error(t, err)

	require.NoError(t, s3.Create(ctx, 5, bytes.NewBufferString("s3 log")))
	require.Equal(t, []byte("s3 log"), objects["/bucket/logs/prefix/5"])

	rc, err := s3.Find(ctx, 5)
	require.NoError(t, err)
	require.Equal(t, "s3 log", rstReadAll(t, rc))

	require.NoError(t, s3.Update(ctx, 5, bytes.NewBufferString("updated log")))
	rc, err = s3.Find(ctx, 5)
	require.NoError(t, err)
	require.Equal(t, "updated log", rstReadAll(t, rc))

	require.NoError(t, s3.Delete(ctx, 5))
	require.Empty(t, objects)
}

func TestRstProvideLogStore(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	t.Setenv("AWS_REGION", "us-east-1")

	ctx := context.Background()
	db := rstNewDB(t)

	t.Run("database only", func(t *testing.T) {
		logStore := logs.ProvideLogStore(db, &types.Config{})
		require.NoError(t, logStore.Create(ctx, 9, strings.NewReader("db log")))

		rc, err := logStore.Find(ctx, 9)
		require.NoError(t, err)
		require.Equal(t, "db log", rstReadAll(t, rc))
	})

	t.Run("s3 with database fallback", func(t *testing.T) {
		srv, objects := rstFakeS3(t)
		cfg := &types.Config{}
		cfg.Logs.S3.Bucket = "bucket"
		cfg.Logs.S3.Prefix = "prefix"
		cfg.Logs.S3.Endpoint = srv.URL
		cfg.Logs.S3.PathStyle = true

		logStore := logs.ProvideLogStore(db, cfg)

		// logs of older builds still live in the database.
		rc, err := logStore.Find(ctx, 9)
		require.NoError(t, err)
		require.Equal(t, "db log", rstReadAll(t, rc))

		// new logs are written to s3.
		require.NoError(t, logStore.Create(ctx, 10, strings.NewReader("s3 log")))
		require.Equal(t, []byte("s3 log"), objects["/bucket/prefix/10"])
	})
}
