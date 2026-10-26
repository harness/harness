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
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFiles(t *testing.T) {
	requireGit(t)

	t.Run("empty repository", func(t *testing.T) {
		ctx := t.Context()
		s := newTestSharedRepo(ctx, t, newTestBareRepo(t))

		// The freshly initialized shared repository has only objects/info/alternates,
		// and that file must not be reported.
		files, err := s.Files()
		require.NoError(t, err)
		assert.Empty(t, files)
	})

	t.Run("objects info directory is excluded", func(t *testing.T) {
		ctx := t.Context()
		s := newTestSharedRepo(ctx, t, newTestBareRepo(t))

		infoDir := filepath.Join(s.Directory(), "objects", "info")
		require.NoError(t, os.WriteFile(filepath.Join(infoDir, "packs"), []byte("P pack-abc.pack\n"), 0o600))

		objectSHAs := writeTestObjects(t, s, 2)

		files, err := s.Files()
		require.NoError(t, err)
		assert.ElementsMatch(t, looseObjectPaths(objectSHAs), files)
	})

	t.Run("loose objects", func(t *testing.T) {
		ctx := t.Context()
		s := newTestSharedRepo(ctx, t, newTestBareRepo(t))

		objectSHAs := writeTestObjects(t, s, 5)

		files, err := s.Files()
		require.NoError(t, err)

		// paths are relative to the objects directory and use forward slashes: "ab/cdef..."
		assert.ElementsMatch(t, looseObjectPaths(objectSHAs), files)
		for _, f := range files {
			assert.Regexp(t, reLooseObject, f)
		}

		// listing the files is read only: nothing is moved or removed, so a second call returns the same list.
		for _, f := range files {
			assert.FileExists(t, filepath.Join(s.Directory(), "objects", filepath.FromSlash(f)))
		}

		filesAgain, err := s.Files()
		require.NoError(t, err)
		assert.Equal(t, files, filesAgain)
	})

	t.Run("pack files", func(t *testing.T) {
		ctx := t.Context()
		s := newTestSharedRepo(ctx, t, newTestBareRepo(t))

		writeTestObjects(t, s, 5)
		require.NoError(t, s.PackObjects(ctx, 1))

		files, err := s.Files()
		require.NoError(t, err)

		// the loose objects were replaced by the pack file and its index (and optionally the reverse index).
		packIdx, idxIdx := -1, -1
		for i, f := range files {
			assert.True(t, strings.HasPrefix(f, "pack/pack-"), "unexpected file %q", f)
			assert.NotRegexp(t, reLooseObject, f)

			switch path.Ext(f) {
			case ".pack":
				assert.Equal(t, -1, packIdx, "more than one pack file")
				packIdx = i
			case ".idx":
				assert.Equal(t, -1, idxIdx, "more than one pack index")
				idxIdx = i
			}
		}

		require.NotEqual(t, -1, packIdx, "pack file is missing: %v", files)
		require.NotEqual(t, -1, idxIdx, "pack index is missing: %v", files)

		// git expects the pack file to be in place before its index.
		assert.Less(t, packIdx, idxIdx, "the pack file must come before the pack index: %v", files)
		requireNonDecreasingPriority(t, files)
	})

	t.Run("loose objects come before pack files", func(t *testing.T) {
		ctx := t.Context()
		s := newTestSharedRepo(ctx, t, newTestBareRepo(t))

		writeTestObjects(t, s, 5)
		require.NoError(t, s.PackObjects(ctx, 1))

		// objects written after packing stay loose
		looseSHAs := []string{
			writeTestLooseObject(t, s, "loose object written after packing 1\n"),
			writeTestLooseObject(t, s, "loose object written after packing 2\n"),
			writeTestLooseObject(t, s, "loose object written after packing 3\n"),
		}

		files, err := s.Files()
		require.NoError(t, err)

		looseCount := len(looseSHAs)
		require.Greater(t, len(files), looseCount, "expected pack files after the loose objects: %v", files)

		assert.ElementsMatch(t, looseObjectPaths(looseSHAs), files[:looseCount])
		for _, f := range files[looseCount:] {
			assert.True(t, strings.HasPrefix(f, "pack/pack-"), "expected a pack file, got %q", f)
		}

		requireNonDecreasingPriority(t, files)
	})

	t.Run("same files and order as MoveObjects", func(t *testing.T) {
		ctx := t.Context()
		sourceRepoPath := newTestBareRepo(t)
		s := newTestSharedRepo(ctx, t, sourceRepoPath)

		// a mix of pack files and several loose objects (with the same priority) to exercise the ordering.
		writeTestObjects(t, s, 5)
		require.NoError(t, s.PackObjects(ctx, 1))
		for _, content := range []string{"a\n", "b\n", "c\n", "d\n", "e\n", "f\n"} {
			writeTestLooseObject(t, s, "loose object "+content)
		}

		files, err := s.Files()
		require.NoError(t, err)
		require.NotEmpty(t, files)

		moved, err := s.MoveObjects(ctx)
		require.NoError(t, err)

		assert.Equal(t, moved, files)
	})

	t.Run("missing objects directory", func(t *testing.T) {
		ctx := t.Context()
		s := newTestSharedRepo(ctx, t, newTestBareRepo(t))

		require.NoError(t, os.RemoveAll(filepath.Join(s.Directory(), "objects")))

		files, err := s.Files()
		require.Error(t, err)
		assert.Nil(t, files)
	})
}

// looseObjectPaths returns the path of the loose object file for each of the object names,
// relative to the object database: "ab/cdef...".
func looseObjectPaths(objectSHAs []string) []string {
	paths := make([]string, len(objectSHAs))
	for i, objectSHA := range objectSHAs {
		paths[i] = objectSHA[:2] + "/" + objectSHA[2:]
	}

	return paths
}

// writeTestLooseObject writes a single blob with the provided content to the shared repository's
// object database and returns its name.
func writeTestLooseObject(t *testing.T, s *SharedRepo, content string) string {
	t.Helper()

	return runTestGit(t, s.Directory(), strings.NewReader(content), "hash-object", "-w", "--stdin")
}

// requireNonDecreasingPriority checks that the files are in the order in which they should be moved:
// loose objects first and the pack files after them (the pack file, then the index).
func requireNonDecreasingPriority(t *testing.T, files []string) {
	t.Helper()

	for i := 1; i < len(files); i++ {
		prev, curr := filePriority(path.Base(files[i-1])), filePriority(path.Base(files[i]))
		require.LessOrEqual(t, prev, curr,
			"%q must not come before %q: %v", files[i-1], files[i], files)
	}
}
