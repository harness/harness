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

package storage

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGtxLocalStore_Save(t *testing.T) {
	store := NewLocalStore()

	t.Run("creates parent directories", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "a", "b", "file.bin")

		got, err := store.Save(path, strings.NewReader("payload"))
		require.NoError(t, err)
		assert.Equal(t, path, got)

		content, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "payload", string(content))
	})

	t.Run("overwrites an existing file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "file.bin")
		require.NoError(t, os.WriteFile(path, []byte("a much longer previous content"), 0o600))

		_, err := store.Save(path, strings.NewReader("new"))
		require.NoError(t, err)

		content, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "new", string(content))
	})

	t.Run("empty data", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "empty")

		_, err := store.Save(path, strings.NewReader(""))
		require.NoError(t, err)

		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Zero(t, info.Size())
	})

	t.Run("directory cannot be created", func(t *testing.T) {
		blocker := filepath.Join(t.TempDir(), "blocker")
		require.NoError(t, os.WriteFile(blocker, nil, 0o600))

		_, err := store.Save(filepath.Join(blocker, "sub", "file"), strings.NewReader("x"))
		require.Error(t, err)
	})

	t.Run("target is a directory", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "dir")
		require.NoError(t, os.MkdirAll(dir, 0o700))

		_, err := store.Save(dir, strings.NewReader("x"))
		require.ErrorContains(t, err, "cannot create file")
	})

	t.Run("read error", func(t *testing.T) {
		readErr := errors.New("gtx read failure")

		_, err := store.Save(filepath.Join(t.TempDir(), "file"), iotest.ErrReader(readErr))
		require.ErrorIs(t, err, readErr)
		require.ErrorContains(t, err, "cannot write to file")
	})
}

func TestGtxProvideLocalStore(t *testing.T) {
	s := ProvideLocalStore()
	require.NotNil(t, s)
	assert.IsType(t, &LocalStore{}, s)
}
