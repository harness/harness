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

package directory

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGtxCreateSubdir(t *testing.T) {
	root := t.TempDir()

	got, err := CreateSubdir(root, "a/b")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(root, "a", "b"), got)

	info, err := os.Stat(got)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
	assert.Equal(t, os.FileMode(FileMode700), info.Mode().Perm())

	// creating an existing directory is fine.
	again, err := CreateSubdir(root, "a/b")
	require.NoError(t, err)
	assert.Equal(t, got, again)

	// a file in the way makes the creation fail.
	file := filepath.Join(root, "file")
	require.NoError(t, os.WriteFile(file, nil, 0o600))
	_, err = CreateSubdir(file, "sub")
	require.Error(t, err)
}

func TestGtxTemporaryPath(t *testing.T) {
	t.Run("create in base dir and remove", func(t *testing.T) {
		base := filepath.Join(t.TempDir(), "x", "y")

		path, err := CreateTemporaryPath(base, "gtx")
		require.NoError(t, err)
		assert.DirExists(t, path)
		assert.Equal(t, base, filepath.Dir(path))
		assert.Contains(t, filepath.Base(path), "gtx")
		assert.Contains(t, filepath.Base(path), ".git")

		require.NoError(t, os.WriteFile(filepath.Join(path, "f"), []byte("x"), 0o600))

		require.NoError(t, RemoveTemporaryPath(path))
		assert.NoDirExists(t, path)

		// removing a path that doesn't exist is a no-op.
		require.NoError(t, RemoveTemporaryPath(path))
	})

	t.Run("empty base dir uses system temp dir", func(t *testing.T) {
		path, err := CreateTemporaryPath("", "gtx")
		require.NoError(t, err)
		t.Cleanup(func() { _ = os.RemoveAll(path) })

		assert.DirExists(t, path)
		assert.Equal(t, filepath.Clean(os.TempDir()), filepath.Dir(path))
	})

	t.Run("base dir cannot be created", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "file")
		require.NoError(t, os.WriteFile(file, nil, 0o600))

		_, err := CreateTemporaryPath(filepath.Join(file, "sub"), "gtx")
		require.Error(t, err)
	})

	t.Run("temp dir cannot be created", func(t *testing.T) {
		_, err := CreateTemporaryPath("", "gtx/../../nonexistent-parent-gtx/x")
		require.Error(t, err)
	})
}
