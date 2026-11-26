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

package maintenance

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gtxTouch(t *testing.T, path string, age time.Duration) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte("x"), 0o600))

	mtime := time.Now().Add(-age)
	require.NoError(t, os.Chtimes(path, mtime, mtime))
}

func TestGtxFindTempObjects(t *testing.T) {
	repo := t.TempDir()
	objects := filepath.Join(repo, "objects")

	old := 48 * time.Hour
	recent := time.Hour

	staleLoose := filepath.Join(objects, "ab", "tmp_obj_stale")
	stalePack := filepath.Join(objects, "pack", "tmp_pack_stale")
	staleNested := filepath.Join(objects, "pack", "nested", "tmp_idx_stale")

	gtxTouch(t, staleLoose, old)
	gtxTouch(t, stalePack, old)
	gtxTouch(t, staleNested, old)
	gtxTouch(t, filepath.Join(objects, "ab", "tmp_obj_recent"), recent)
	gtxTouch(t, filepath.Join(objects, "pack", "pack-1.pack"), old)
	gtxTouch(t, filepath.Join(objects, "cd", "not_tmp_old"), old)
	gtxTouch(t, filepath.Join(repo, "tmp_outside_objects"), old)

	got, err := FindTempObjects(repo)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{staleLoose, stalePack, staleNested}, got)
}

func TestGtxFindTempObjects_Boundaries(t *testing.T) {
	t.Run("missing repository", func(t *testing.T) {
		got, err := FindTempObjects(filepath.Join(t.TempDir(), "does-not-exist"))
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("empty objects directory", func(t *testing.T) {
		repo := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(repo, "objects"), 0o700))

		got, err := FindTempObjects(repo)
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("age threshold", func(t *testing.T) {
		repo := t.TempDir()
		objects := filepath.Join(repo, "objects")

		justOver := filepath.Join(objects, "tmp_just_over")
		justUnder := filepath.Join(objects, "tmp_just_under")
		gtxTouch(t, justOver, deleteTempFilesOlderThanDuration+time.Minute)
		gtxTouch(t, justUnder, deleteTempFilesOlderThanDuration-time.Minute)

		got, err := FindTempObjects(repo)
		require.NoError(t, err)
		assert.Equal(t, []string{justOver}, got)
	})

	t.Run("objects is a file", func(t *testing.T) {
		repo := t.TempDir()
		gtxTouch(t, filepath.Join(repo, "objects"), 48*time.Hour)

		got, err := FindTempObjects(repo)
		require.NoError(t, err)
		assert.Empty(t, got, "a plain file named objects isn't a temp object")
	})

	t.Run("unreadable subdirectory is skipped", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("permissions aren't enforced for root")
		}

		repo := t.TempDir()
		objects := filepath.Join(repo, "objects")

		visible := filepath.Join(objects, "tmp_visible")
		gtxTouch(t, visible, 48*time.Hour)

		locked := filepath.Join(objects, "locked")
		gtxTouch(t, filepath.Join(locked, "tmp_hidden"), 48*time.Hour)
		require.NoError(t, os.Chmod(locked, 0o000))
		t.Cleanup(func() { _ = os.Chmod(locked, 0o700) })

		got, err := FindTempObjects(repo)
		require.NoError(t, err)
		assert.Equal(t, []string{visible}, got)
	})
}
