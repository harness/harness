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
	"io/fs"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestFS(t *testing.T) *FS {
	t.Helper()
	requireGit(t)
	repo := initRepoWithCommit(t)
	writeRepoFile(t, repo, "dir/inner.txt", "inner\n")
	writeRepoFile(t, repo, "dir/sub/deep.md", "deep\n")
	writeRepoFile(t, repo, "run.sh", "#!/bin/sh\n")
	require.NoError(t, chmodExec(repo, "run.sh"))
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "tree")
	return NewFS(t.Context(), "main", repo)
}

func TestFS_ReadFileAndStat(t *testing.T) {
	gfs := newTestFS(t)

	data, err := gfs.ReadFile("dir/inner.txt")
	require.NoError(t, err)
	require.Equal(t, "inner\n", string(data))

	info, err := gfs.Stat("dir/inner.txt")
	require.NoError(t, err)
	require.False(t, info.IsDir())
	require.Equal(t, int64(6), info.Size())
	require.Equal(t, fs.FileMode(0o444), info.Mode())
	require.True(t, info.ModTime().IsZero())
	require.Nil(t, info.Sys())

	info, err = gfs.Stat("run.sh")
	require.NoError(t, err)
	require.Equal(t, fs.FileMode(0o555), info.Mode())

	info, err = gfs.Stat("dir")
	require.NoError(t, err)
	require.True(t, info.IsDir())
	require.True(t, info.Mode().IsDir())

	_, err = gfs.Stat("missing")
	require.Error(t, err)

	_, err = gfs.ReadFile("missing")
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestFS_Open(t *testing.T) {
	gfs := newTestFS(t)

	t.Run("invalid path", func(t *testing.T) {
		_, err := gfs.Open("../escape")
		require.ErrorIs(t, err, fs.ErrInvalid)
	})

	t.Run("file", func(t *testing.T) {
		f, err := gfs.Open("dir/inner.txt")
		require.NoError(t, err)
		defer f.Close()

		st, err := f.Stat()
		require.NoError(t, err)
		require.Equal(t, "dir/inner.txt", st.Name())
		require.False(t, st.IsDir())
		require.True(t, st.ModTime().IsZero())
		require.Nil(t, st.Sys())

		data, err := io.ReadAll(f)
		require.NoError(t, err)
		require.Equal(t, "inner\n", string(data))
	})

	t.Run("directory", func(t *testing.T) {
		f, err := gfs.Open("dir")
		require.NoError(t, err)
		defer f.Close()

		st, err := f.Stat()
		require.NoError(t, err)
		require.True(t, st.IsDir())
		require.Equal(t, fs.ModeDir, st.Mode())
		require.Zero(t, st.Size())
		require.True(t, st.ModTime().IsZero())
		require.Nil(t, st.Sys())

		_, err = f.Read(make([]byte, 1))
		require.ErrorIs(t, err, fs.ErrInvalid)

		dir, ok := f.(fs.ReadDirFile)
		require.True(t, ok)

		first, err := dir.ReadDir(1)
		require.NoError(t, err)
		require.Len(t, first, 1)
		rest, err := dir.ReadDir(-1)
		require.NoError(t, err)
		require.Len(t, rest, 1)
		_, err = dir.ReadDir(1)
		require.ErrorIs(t, err, io.EOF)
	})

	t.Run("missing", func(t *testing.T) {
		_, err := gfs.Open("nope.txt")
		require.ErrorIs(t, err, fs.ErrNotExist)
	})
}

func TestFS_ReadDirAndGlob(t *testing.T) {
	gfs := newTestFS(t)

	entries, err := gfs.ReadDir("dir")
	require.NoError(t, err)
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
		info, err := e.Info()
		require.NoError(t, err)
		require.Equal(t, e.IsDir(), info.IsDir())
		require.Equal(t, info.Mode(), e.Type())
		require.True(t, info.ModTime().IsZero())
		require.Nil(t, info.Sys())
	}
	sort.Strings(names)
	require.Equal(t, []string{"inner.txt", "sub"}, names)

	matches, err := gfs.Glob("dir/*.txt")
	require.NoError(t, err)
	require.Equal(t, []string{"dir/inner.txt"}, matches)

	matches, err = gfs.Glob("dir/**/*.md")
	require.NoError(t, err)
	require.Equal(t, []string{"dir/sub/deep.md"}, matches)

	_, err = gfs.ReadDir("missing")
	require.Error(t, err)
}
