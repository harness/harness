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

package sharedrepo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackObjects(t *testing.T) {
	requireGit(t)

	tests := []struct {
		name        string
		objectCount int
		wantPacked  bool
	}{
		{
			name:        "test below the threshold",
			objectCount: packObjectsThreshold - 1,
			wantPacked:  false,
		},
		{
			name:        "test at the threshold",
			objectCount: packObjectsThreshold,
			wantPacked:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			sourceRepoPath := newTestBareRepo(t)
			s := newTestSharedRepo(ctx, t, sourceRepoPath)

			objectSHAs := writeTestObjects(t, s, test.objectCount)

			require.NoError(t, s.PackObjects(ctx))

			looseCount, packCount := countObjectFiles(t, s)
			if test.wantPacked {
				assert.Zero(t, looseCount, "loose objects should have been replaced by the pack file")
				assert.Equal(t, 1, packCount)
			} else {
				assert.Equal(t, test.objectCount, looseCount)
				assert.Zero(t, packCount)
			}

			// no matter how they are stored, all objects must still be readable
			requireObjectsExist(t, s.Directory(), objectSHAs)

			// and they must survive the move to the original repository
			require.NoError(t, s.MoveObjects(ctx))

			requireObjectsExist(t, sourceRepoPath, objectSHAs)

			// the object database of the original repository must be left consistent
			runTestGit(t, sourceRepoPath, nil, "fsck", "--strict")
		})
	}
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available on PATH")
	}
}

// newTestBareRepo creates an empty bare git repository.
func newTestBareRepo(t *testing.T) string {
	t.Helper()

	repoPath := t.TempDir()
	runTestGit(t, repoPath, nil, "init", "--bare")

	return repoPath
}

// newTestSharedRepo creates a shared repository with the given repository as its object source.
func newTestSharedRepo(ctx context.Context, t *testing.T, repoPath string) *SharedRepo {
	t.Helper()

	s, err := NewSharedRepo(t.TempDir(), repoPath)
	require.NoError(t, err)
	t.Cleanup(func() { s.Close(ctx) })
	require.NoError(t, s.Init(ctx))

	return s
}

// writeTestObjects writes the requested number of blobs to the shared repository's
// object database and returns their names.
func writeTestObjects(t *testing.T, s *SharedRepo, count int) []string {
	t.Helper()

	dir := t.TempDir()
	paths := bytes.NewBuffer(nil)

	for i := range count {
		filePath := filepath.Join(dir, fmt.Sprintf("file%d", i))
		content := fmt.Sprintf("content of the blob number %d\n", i)
		require.NoError(t, os.WriteFile(filePath, []byte(content), 0o600))
		paths.WriteString(filePath + "\n")
	}

	// hash all files with a single git process to keep the test fast
	out := runTestGit(t, s.Directory(), paths, "hash-object", "-w", "--stdin-paths")

	objectSHAs := strings.Fields(out)
	require.Len(t, objectSHAs, count)

	return objectSHAs
}

// requireObjectsExist checks that all objects are blobs available in the given repository.
func requireObjectsExist(t *testing.T, repoPath string, objectSHAs []string) {
	t.Helper()

	stdin := bytes.NewBufferString(strings.Join(objectSHAs, "\n") + "\n")

	// a single git process checks all objects at once to keep the test fast
	out := runTestGit(t, repoPath, stdin, "cat-file", "--batch-check")

	lines := strings.Split(out, "\n")
	require.Len(t, lines, len(objectSHAs))

	for i, objectSHA := range objectSHAs {
		assert.True(t, strings.HasPrefix(lines[i], objectSHA+" blob "),
			"object %s is not available: %q", objectSHA, lines[i])
	}
}

// countObjectFiles returns the number of loose objects and of pack files
// in the shared repository's object database.
func countObjectFiles(t *testing.T, s *SharedRepo) (looseCount, packCount int) {
	t.Helper()

	files, err := s.objectFiles()
	require.NoError(t, err)

	for _, f := range files {
		switch {
		case reLooseObject.MatchString(f.relPath):
			looseCount++
		case strings.HasSuffix(f.fileName, ".pack"):
			packCount++
		}
	}

	return looseCount, packCount
}

func runTestGit(t *testing.T, repoPath string, stdin io.Reader, args ...string) string {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), "git", args...)
	cmd.Dir = repoPath
	cmd.Stdin = stdin

	out, err := cmd.Output()
	require.NoErrorf(t, err, "git %s failed", strings.Join(args, " "))

	return strings.TrimSpace(string(out))
}
