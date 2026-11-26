//  Copyright 2023 Harness, Inc.
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

package filesystem

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	storagedriver "github.com/harness/gitness/registry/app/driver"

	"github.com/stretchr/testify/require"
)

// rb5Inner returns the unwrapped driver rooted in a temporary directory.
func rb5Inner(t *testing.T) (*driver, string) {
	t.Helper()
	root := t.TempDir()
	return &driver{rootDirectory: root}, root
}

// rb5BlockedPath creates a regular file and returns a sub path below it, which no operation can resolve.
func rb5BlockedPath(t *testing.T, root string) string {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(root, "blocker"), []byte("x"), 0o600))
	return "/blocker/child"
}

func TestRb5FromParametersAndFactory(t *testing.T) {
	ctx := context.Background()

	Register(ctx)

	created, err := (&filesystemDriverFactory{}).Create(ctx, map[string]any{"rootdirectory": t.TempDir()})
	require.NoError(t, err)
	require.Equal(t, driverName, created.Name())

	_, err = FromParameters(map[string]any{"maxthreads": "not-a-number"})
	require.ErrorContains(t, err, "maxthreads config error")

	defaults, err := FromParameters(nil)
	require.NoError(t, err)
	require.NotNil(t, defaults)
}

func TestRb5CopyObject(t *testing.T) {
	ctx := context.Background()
	d, root := rb5Inner(t)
	require.NoError(t, d.PutContent(ctx, "/src/file", []byte("payload")))

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	require.ErrorIs(t, d.CopyObject(cancelled, "/src/file", "", "/dst/file"), context.Canceled)

	require.ErrorAs(t, d.CopyObject(ctx, "/src/missing", "", "/dst/file"), &storagedriver.PathNotFoundError{})
	require.ErrorContains(t, d.CopyObject(ctx, "/src", "", "/dst/file"), "source is a directory")

	require.NoError(t, d.CopyObject(ctx, "/src/file", "", "/dst/nested/file"))
	got, err := d.GetContent(ctx, "/dst/nested/file")
	require.NoError(t, err)
	require.Equal(t, "payload", string(got))

	// Overwriting an existing copy works.
	require.NoError(t, d.CopyObject(ctx, "/src/file", "", "/dst/nested/file"))

	// The destination parent cannot be created below a regular file.
	require.Error(t, d.CopyObject(ctx, "/src/file", "", rb5BlockedPath(t, root)))

	// The destination is an existing directory, so the final rename fails.
	require.NoError(t, d.PutContent(ctx, "/occupied/inner", []byte("x")))
	require.Error(t, d.CopyObject(ctx, "/src/file", "", "/occupied"))
}

func TestRb5BatchCopyObjects(t *testing.T) {
	ctx := context.Background()
	d, _ := rb5Inner(t)
	require.NoError(t, d.PutContent(ctx, "/batch/a", []byte("a")))
	require.NoError(t, d.PutContent(ctx, "/batch/b", []byte("b")))

	require.NoError(t, d.BatchCopyObjects(ctx, "", []string{"/batch/a", "/batch/b"}, 0))
	require.NoError(t, d.BatchCopyObjects(ctx, "", nil, 2))

	err := d.BatchCopyObjects(ctx, "", []string{"/batch/a", "/batch/missing"}, 2)
	require.ErrorContains(t, err, "failed to copy key /batch/missing after 3 retries")
}

func TestRb5CopyFileContents(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	require.NoError(t, os.WriteFile(src, []byte("abc"), 0o600))

	dst := filepath.Join(dir, "dst")
	require.NoError(t, copyFileContents(ctx, src, dst, 0o640))
	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	require.Equal(t, "abc", string(got))

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	require.ErrorIs(t, copyFileContents(cancelled, src, dst, 0o640), context.Canceled)
	require.Error(t, copyFileContents(ctx, filepath.Join(dir, "missing"), dst, 0o640))
	require.Error(t, copyFileContents(ctx, src, filepath.Join(dir, "no", "such", "dir"), 0o640))
}

func TestRb5ContentAndReaderErrors(t *testing.T) {
	ctx := context.Background()
	d, root := rb5Inner(t)
	blocked := rb5BlockedPath(t, root)

	_, err := d.GetContent(ctx, "/missing")
	require.ErrorAs(t, err, &storagedriver.PathNotFoundError{})
	_, err = d.Reader(ctx, "/missing", 0)
	require.ErrorAs(t, err, &storagedriver.PathNotFoundError{})
	_, err = d.Reader(ctx, blocked, 0)
	require.Error(t, err)

	require.NoError(t, d.PutContent(ctx, "/file", []byte("0123456789")))
	_, err = d.Reader(ctx, "/file", -1)
	require.Error(t, err)

	rc, err := d.Reader(ctx, "/file", 4)
	require.NoError(t, err)
	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.NoError(t, rc.Close())
	require.Equal(t, "456789", string(got))

	// Reading a directory fails while draining the file.
	require.NoError(t, os.MkdirAll(filepath.Join(root, "adir"), 0o755))
	_, err = d.GetContent(ctx, "/adir")
	require.Error(t, err)

	require.Error(t, d.PutContent(ctx, blocked, []byte("x")))
}

func TestRb5Writer(t *testing.T) {
	ctx := context.Background()
	d, root := rb5Inner(t)

	_, err := d.Writer(ctx, rb5BlockedPath(t, root), false)
	require.Error(t, err)

	// A directory cannot be opened for writing.
	require.NoError(t, os.MkdirAll(filepath.Join(root, "adir"), 0o755))
	_, err = d.Writer(ctx, "/adir", false)
	require.Error(t, err)

	w, err := d.Writer(ctx, "/w/file", false)
	require.NoError(t, err)
	_, err = w.Write([]byte("abc"))
	require.NoError(t, err)
	require.Equal(t, int64(3), w.Size())
	require.NoError(t, w.Commit(ctx))
	require.NoError(t, w.Close())

	appended, err := d.Writer(ctx, "/w/file", true)
	require.NoError(t, err)
	require.Equal(t, int64(3), appended.Size())
	_, err = appended.Write([]byte("def"))
	require.NoError(t, err)
	require.NoError(t, appended.Close())

	got, err := d.GetContent(ctx, "/w/file")
	require.NoError(t, err)
	require.Equal(t, "abcdef", string(got))

	truncated, err := d.Writer(ctx, "/w/file", false)
	require.NoError(t, err)
	require.Zero(t, truncated.Size())
	require.NoError(t, truncated.Close())
}

func TestRb5FileWriterStates(t *testing.T) {
	ctx := context.Background()
	d, _ := rb5Inner(t)

	newWriter := func(t *testing.T) storagedriver.FileWriter {
		t.Helper()
		w, err := d.Writer(ctx, "/states/file", false)
		require.NoError(t, err)
		return w
	}

	closed := newWriter(t)
	require.NoError(t, closed.Close())
	require.ErrorContains(t, closed.Close(), "already closed")
	_, err := closed.Write([]byte("x"))
	require.ErrorContains(t, err, "already closed")
	require.ErrorContains(t, closed.Commit(ctx), "already closed")
	require.ErrorContains(t, closed.Cancel(ctx), "already closed")

	committed := newWriter(t)
	require.NoError(t, committed.Commit(ctx))
	_, err = committed.Write([]byte("x"))
	require.ErrorContains(t, err, "already committed")
	require.ErrorContains(t, committed.Commit(ctx), "already committed")
	require.NoError(t, committed.Close())

	cancelled := newWriter(t)
	require.NoError(t, cancelled.Cancel(ctx))
	_, err = cancelled.Write([]byte("x"))
	require.ErrorContains(t, err, "already cancelled")
	require.ErrorContains(t, cancelled.Commit(ctx), "already cancelled")
}

func TestRb5StatListMoveDelete(t *testing.T) {
	ctx := context.Background()
	d, root := rb5Inner(t)
	blocked := rb5BlockedPath(t, root)
	require.NoError(t, d.PutContent(ctx, "/tree/a", []byte("aa")))
	require.NoError(t, d.PutContent(ctx, "/tree/b", []byte("b")))

	// Stat.
	_, err := d.Stat(ctx, "/tree/missing")
	require.ErrorAs(t, err, &storagedriver.PathNotFoundError{})
	_, err = d.Stat(ctx, blocked)
	require.Error(t, err)
	fi, err := d.Stat(ctx, "/tree/a")
	require.NoError(t, err)
	require.Equal(t, "/tree/a", fi.Path())
	require.Equal(t, int64(2), fi.Size())
	require.False(t, fi.IsDir())
	require.WithinDuration(t, time.Now(), fi.ModTime(), time.Minute)
	dir, err := d.Stat(ctx, "/tree")
	require.NoError(t, err)
	require.True(t, dir.IsDir())
	require.Zero(t, dir.Size())

	// List.
	_, err = d.List(ctx, "/missing")
	require.ErrorAs(t, err, &storagedriver.PathNotFoundError{})
	_, err = d.List(ctx, blocked)
	require.Error(t, err)
	_, err = d.List(ctx, "/tree/a")
	require.Error(t, err)
	keys, err := d.List(ctx, "/tree")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"/tree/a", "/tree/b"}, keys)

	// Move.
	require.ErrorAs(t, d.Move(ctx, "/tree/missing", "/other/x"), &storagedriver.PathNotFoundError{})
	require.Error(t, d.Move(ctx, "/tree/a", blocked))
	require.NoError(t, d.PutContent(ctx, "/full/inner", []byte("x")))
	require.Error(t, d.Move(ctx, "/tree/a", "/full"))
	require.NoError(t, d.Move(ctx, "/tree/a", "/moved/a"))
	_, err = d.Stat(ctx, "/tree/a")
	require.Error(t, err)

	// Delete.
	require.ErrorAs(t, d.Delete(ctx, "/missing"), &storagedriver.PathNotFoundError{})
	require.Error(t, d.Delete(ctx, blocked))
	require.NoError(t, d.Delete(ctx, "/tree"))
	_, err = d.Stat(ctx, "/tree")
	require.ErrorAs(t, err, &storagedriver.PathNotFoundError{})
}

func TestRb5RedirectAndWalk(t *testing.T) {
	ctx := context.Background()
	d, _ := rb5Inner(t)
	require.NoError(t, d.PutContent(ctx, "/walk/a", []byte("a")))
	require.NoError(t, d.PutContent(ctx, "/walk/sub/b", []byte("b")))

	url, err := d.RedirectURL(ctx, "GET", "/walk/a", "")
	require.NoError(t, err)
	require.Empty(t, url)

	var paths []string
	require.NoError(t, d.Walk(ctx, "/walk", func(fi storagedriver.FileInfo) error {
		paths = append(paths, fi.Path())
		return nil
	}))
	require.Contains(t, paths, "/walk/a")
	require.Contains(t, paths, "/walk/sub/b")
}
