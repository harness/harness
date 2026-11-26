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

package api

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/git/sha"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

func TestGc5RedisLastCommitCache(t *testing.T) {
	c, err := NewRedisLastCommitCache(nil, time.Minute)
	require.Error(t, err)
	require.Nil(t, c)

	client := gc5RedisClient()
	t.Cleanup(func() { _ = client.Close() })

	c, err = NewRedisLastCommitCache(client, time.Minute)
	require.NoError(t, err)
	require.NotNil(t, c)

	// must not panic.
	logCacheErrFn(t.Context(), io.ErrUnexpectedEOF)
}

func TestGc5NoLastCommitCache(t *testing.T) {
	c := NoLastCommitCache()

	hits, misses := c.Stats()
	require.Zero(t, hits)
	require.Zero(t, misses)

	// evicting from a cache that does not cache is a no-op.
	c.Evict(t.Context(), CommitEntryKey("k"))
}

func TestGc5GitFailuresOutsideRepository(t *testing.T) {
	notARepo := t.TempDir()
	g := &Git{}
	ctx := t.Context()

	tests := []struct {
		name string
		run  func() error
	}{
		{"branch count", func() error { _, err := g.GetBranchCount(ctx, notARepo); return err }},
		{"tag count", func() error { _, err := g.GetTagCount(ctx, notARepo); return err }},
		{"hash object", func() error {
			_, err := g.HashObject(ctx, notARepo, strings.NewReader("content"))
			return err
		}},
		{"set default branch", func() error { return g.SetDefaultBranch(ctx, notARepo, "main", true) }},
		{"submodule", func() error { _, err := g.GetSubmodule(ctx, notARepo, "main", "sub"); return err }},
		{"blob sha for paths", func() error {
			_, err := g.GetBlobSHAsForPaths(ctx, notARepo, nil, "main", []string{"a.txt"})
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Error(t, tt.run())
		})
	}
}

func TestGc5GitFailuresOnMissingObjects(t *testing.T) {
	repo, _, _ := gc5Repo(t)
	ctx := t.Context()
	unknownSHA := sha.Must(strings.Repeat("1", 40))

	_, err := (&Git{}).GetCommits(ctx, repo, []sha.SHA{unknownSHA})
	require.Error(t, err)

	_, err = CatFileAnnotatedTagFromSHAs(ctx, repo, nil, []sha.SHA{unknownSHA})
	require.Error(t, err)

	// setting a default branch that doesn't exist is refused unless requested otherwise.
	require.Error(t, (&Git{}).SetDefaultBranch(ctx, repo, "does-not-exist", false))
	require.NoError(t, (&Git{}).SetDefaultBranch(ctx, repo, "does-not-exist", true))

	_, err = (&Git{}).GetSubmodule(ctx, repo, "main", "missing")
	require.Error(t, err)
}

func TestGc5LoadRepositoryInfo(t *testing.T) {
	dir := t.TempDir()

	_, err := LoadRepositoryInfo(dir)
	require.Error(t, err)

	repo, _, _ := gc5Repo(t)
	info, err := LoadRepositoryInfo(filepath.Join(repo, ".git"))
	require.NoError(t, err)
	require.Positive(t, info.LooseObjects.Count)
	require.Positive(t, info.References.LooseReferenceCount)

	refs, err := LoadReferencesInfo(filepath.Join(dir, "missing"))
	require.NoError(t, err)
	require.Zero(t, refs.LooseReferenceCount)

	require.NoError(t, os.MkdirAll(filepath.Join(dir, "refs", "heads"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "refs", "heads", "x"), []byte("sha"), 0o600))
	refs, err = LoadReferencesInfo(dir)
	require.NoError(t, err)
	require.Equal(t, uint64(1), refs.LooseReferenceCount)
}

func gc5RedisClient() redis.UniversalClient {
	return redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
}
