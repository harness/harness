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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harness/gitness/git/sha"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestSourceRepo creates a bare repository holding a single commit with the given
// files and returns the repository path and the commit SHA.
func newTestSourceRepo(t *testing.T, files map[string]string) (string, sha.SHA) {
	t.Helper()

	workPath := t.TempDir()
	runTestGit(t, workPath, nil, "init", "-b", "main")

	for relPath, content := range files {
		fullPath := filepath.Join(workPath, filepath.FromSlash(relPath))
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o700))
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o600))
	}

	runTestGit(t, workPath, nil, "add", "-A")
	runTestGit(t, workPath, nil,
		"-c", "user.name=test", "-c", "user.email=test@harness.io", "commit", "-m", "init")

	repoPath := t.TempDir()
	runTestGit(t, repoPath, nil, "clone", "--bare", workPath, ".")

	commitSHA, err := sha.New(runTestGit(t, repoPath, nil, "rev-parse", "main"))
	require.NoError(t, err)

	return repoPath, commitSHA
}

// indexPaths returns the recursive listing of the paths in the shared repository's index.
func indexPaths(t *testing.T, s *SharedRepo) []string {
	t.Helper()

	treeSHA, err := s.WriteTree(t.Context())
	require.NoError(t, err)

	out := runTestGit(t, s.Directory(), nil, "ls-tree", "-r", "--name-only", treeSHA.String())
	if out == "" {
		return nil
	}

	return strings.Split(out, "\n")
}

// TestCreateFile_ColonDirectory asserts that a file can be committed into a directory
// named ":".
//
// Regression guard for CODE-6237: file paths are handed to git as pathspecs, and git reads
// a leading ":" as pathspec magic ("from the repository root"). checkPathAvailability
// therefore asked git for the entry ":" and got the whole root listing back, which failed
// the commit with "failed to commit files".
func TestCreateFile_ColonDirectory(t *testing.T) {
	requireGit(t)

	t.Run("new colon directory", func(t *testing.T) {
		ctx := t.Context()

		// more than one root entry is what makes the magic pathspec answer ambiguous
		repoPath, commitSHA := newTestSourceRepo(t, map[string]string{
			"rootfile.txt": "root\n",
			"other/g.txt":  "g\n",
		})

		s := newTestSharedRepo(ctx, t, repoPath)
		require.NoError(t, s.SetIndex(ctx, commitSHA))

		_, err := s.CreateFile(ctx, commitSHA, ":/f.txt", "100644", []byte("f\n"))
		require.NoError(t, err)

		assert.Contains(t, indexPaths(t, s), ":/f.txt")
	})

	t.Run("existing colon directory", func(t *testing.T) {
		ctx := t.Context()

		repoPath, commitSHA := newTestSourceRepo(t, map[string]string{
			"rootfile.txt": "root\n",
			":/f.txt":      "f\n",
		})

		s := newTestSharedRepo(ctx, t, repoPath)
		require.NoError(t, s.SetIndex(ctx, commitSHA))

		_, err := s.CreateFile(ctx, commitSHA, ":/second.txt", "100644", []byte("second\n"))
		require.NoError(t, err)

		assert.Contains(t, indexPaths(t, s), ":/second.txt")
	})

	t.Run("file already exists in colon directory", func(t *testing.T) {
		ctx := t.Context()

		repoPath, commitSHA := newTestSourceRepo(t, map[string]string{
			"rootfile.txt": "root\n",
			":/f.txt":      "f\n",
		})

		s := newTestSharedRepo(ctx, t, repoPath)
		require.NoError(t, s.SetIndex(ctx, commitSHA))

		_, err := s.CreateFile(ctx, commitSHA, ":/f.txt", "100644", []byte("other\n"))
		require.ErrorContains(t, err, "already exists")
	})
}

// TestUpdateFile_ColonDirectory asserts that a file inside a directory named ":" can be
// updated.
//
// Regression guard for CODE-6237.
func TestUpdateFile_ColonDirectory(t *testing.T) {
	requireGit(t)

	ctx := t.Context()

	repoPath, commitSHA := newTestSourceRepo(t, map[string]string{
		"rootfile.txt": "root\n",
		":/f.txt":      "f\n",
	})

	s := newTestSharedRepo(ctx, t, repoPath)
	require.NoError(t, s.SetIndex(ctx, commitSHA))

	blobSHA, err := s.UpdateFile(ctx, commitSHA, ":/f.txt", sha.None, "100644", []byte("updated\n"))
	require.NoError(t, err)

	assert.Contains(t, indexPaths(t, s), ":/f.txt")

	out := runTestGit(t, s.Directory(), nil, "cat-file", "blob", blobSHA.String())
	assert.Equal(t, "updated", out)
}

// TestLsFiles_ColonDirectory asserts that a file inside a directory named ":" is found in the
// index by its literal path.
//
// Regression guard for CODE-6237.
func TestLsFiles_ColonDirectory(t *testing.T) {
	requireGit(t)

	ctx := t.Context()

	repoPath, commitSHA := newTestSourceRepo(t, map[string]string{
		"rootfile.txt": "root\n",
		":/f.txt":      "f\n",
	})

	s := newTestSharedRepo(ctx, t, repoPath)
	require.NoError(t, s.SetIndex(ctx, commitSHA))

	files, err := s.LsFiles(ctx, ":/f.txt")
	require.NoError(t, err)
	assert.Contains(t, files, ":/f.txt")
}
