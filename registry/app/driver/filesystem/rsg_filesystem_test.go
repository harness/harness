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

package filesystem

import (
	"context"
	"errors"
	"io"
	"testing"

	storagedriver "github.com/harness/gitness/registry/app/driver"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const rsgTestPath = "/rsg/data/file.txt"

func rsgNewDriver(t *testing.T) *Driver {
	t.Helper()
	d, err := FromParameters(map[string]any{
		"rootdirectory": t.TempDir(),
		"maxthreads":    uint64(25),
	})
	require.NoError(t, err)
	return d
}

func TestRsg_FilesystemDriverName(t *testing.T) {
	assert.Equal(t, "filesystem", GetDriverName())
	d := rsgNewDriver(t)
	assert.Equal(t, "filesystem", d.Name())
}

func TestRsg_FilesystemPutGetContent(t *testing.T) {
	ctx := context.Background()
	d := rsgNewDriver(t)
	payload := []byte("filesystem payload")

	require.NoError(t, d.PutContent(ctx, rsgTestPath, payload))
	got, err := d.GetContent(ctx, rsgTestPath)
	require.NoError(t, err)
	assert.Equal(t, payload, got)
}

func TestRsg_FilesystemReaderWriter(t *testing.T) {
	ctx := context.Background()
	d := rsgNewDriver(t)
	payload := []byte("stream payload")

	w, err := d.Writer(ctx, "/rsg/stream", false)
	require.NoError(t, err)
	_, err = w.Write(payload)
	require.NoError(t, err)
	require.NoError(t, w.Commit(ctx))
	require.NoError(t, w.Close())

	rc, err := d.Reader(ctx, "/rsg/stream", 0)
	require.NoError(t, err)
	defer rc.Close()
	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, payload, got)
}

func TestRsg_FilesystemWriterCancel(t *testing.T) {
	ctx := context.Background()
	d := rsgNewDriver(t)

	w, err := d.Writer(ctx, "/rsg/cancel", false)
	require.NoError(t, err)
	_, err = w.Write([]byte("x"))
	require.NoError(t, err)
	require.NoError(t, w.Cancel(ctx))
}

func TestRsg_FilesystemStatList(t *testing.T) {
	ctx := context.Background()
	d := rsgNewDriver(t)
	require.NoError(t, d.PutContent(ctx, "/rsg/dir/a", []byte("a")))
	require.NoError(t, d.PutContent(ctx, "/rsg/dir/b", []byte("b")))

	children, err := d.List(ctx, "/rsg/dir")
	require.NoError(t, err)
	assert.Len(t, children, 2)

	fi, err := d.Stat(ctx, "/rsg/dir/a")
	require.NoError(t, err)
	assert.False(t, fi.IsDir())
	assert.Equal(t, int64(1), fi.Size())
	assert.Equal(t, "/rsg/dir/a", fi.Path())
	assert.False(t, fi.ModTime().IsZero())
}

func TestRsg_FilesystemMoveDelete(t *testing.T) {
	ctx := context.Background()
	d := rsgNewDriver(t)
	require.NoError(t, d.PutContent(ctx, "/rsg/src", []byte("mv")))

	require.NoError(t, d.Move(ctx, "/rsg/src", "/rsg/dst"))
	_, err := d.Stat(ctx, "/rsg/src")
	require.Error(t, err)

	require.NoError(t, d.Delete(ctx, "/rsg/dst"))
	_, err = d.Stat(ctx, "/rsg/dst")
	require.Error(t, err)
}

func TestRsg_FilesystemCopyObject(t *testing.T) {
	ctx := context.Background()
	d := rsgNewDriver(t)
	require.NoError(t, d.PutContent(ctx, "/rsg/copy-src", []byte("copy-me")))
	require.NoError(t, d.CopyObject(ctx, "/rsg/copy-src", "", "/rsg/copy-dst"))

	got, err := d.GetContent(ctx, "/rsg/copy-dst")
	require.NoError(t, err)
	assert.Equal(t, []byte("copy-me"), got)
}

func TestRsg_FilesystemCopyMissingSource(t *testing.T) {
	ctx := context.Background()
	d := rsgNewDriver(t)
	err := d.CopyObject(ctx, "/rsg/missing", "", "/rsg/dst")
	require.Error(t, err)
	var pnfe storagedriver.PathNotFoundError
	require.True(t, errors.As(err, &pnfe))
}

func TestRsg_FilesystemWalk(t *testing.T) {
	ctx := context.Background()
	d := rsgNewDriver(t)
	require.NoError(t, d.PutContent(ctx, "/rsg/walk/a", []byte("a")))
	count := 0
	err := d.Walk(ctx, "/rsg/walk", func(fi storagedriver.FileInfo) error {
		if !fi.IsDir() {
			count++
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestRsg_FilesystemRedirectURL(t *testing.T) {
	ctx := context.Background()
	d := rsgNewDriver(t)
	url, err := d.RedirectURL(ctx, "GET", rsgTestPath, "file.txt")
	require.NoError(t, err)
	assert.Empty(t, url)
}

func TestRsg_FromParametersErrors(t *testing.T) {
	_, err := FromParameters(map[string]any{"maxthreads": "not-a-number"})
	require.Error(t, err)
}

func TestRsg_FromParametersDefaults(t *testing.T) {
	d, err := FromParameters(nil)
	require.NoError(t, err)
	require.NotNil(t, d)
}

func TestRsg_FilesystemReaderOffset(t *testing.T) {
	ctx := context.Background()
	d := rsgNewDriver(t)
	require.NoError(t, d.PutContent(ctx, rsgTestPath, []byte("abcdef")))

	rc, err := d.Reader(ctx, rsgTestPath, 3)
	require.NoError(t, err)
	defer rc.Close()
	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, "def", string(got))
}

func TestRsg_FilesystemReaderNotFound(t *testing.T) {
	ctx := context.Background()
	d := rsgNewDriver(t)
	_, err := d.Reader(ctx, "/rsg/missing", 0)
	require.Error(t, err)
	var pnfe storagedriver.PathNotFoundError
	require.True(t, errors.As(err, &pnfe))
}

func TestRsg_FilesystemStatMissing(t *testing.T) {
	ctx := context.Background()
	d := rsgNewDriver(t)
	_, err := d.Stat(ctx, "/rsg/missing")
	require.Error(t, err)
}

func TestRsg_FilesystemDeleteMissing(t *testing.T) {
	ctx := context.Background()
	d := rsgNewDriver(t)
	err := d.Delete(ctx, "/rsg/missing")
	require.Error(t, err)
}

func TestRsg_FilesystemAppendWriter(t *testing.T) {
	ctx := context.Background()
	d := rsgNewDriver(t)
	require.NoError(t, d.PutContent(ctx, "/rsg/append", []byte("ab")))

	w, err := d.Writer(ctx, "/rsg/append", true)
	require.NoError(t, err)
	_, err = w.Write([]byte("cd"))
	require.NoError(t, err)
	require.NoError(t, w.Commit(ctx))
	require.NoError(t, w.Close())

	got, err := d.GetContent(ctx, "/rsg/append")
	require.NoError(t, err)
	assert.Equal(t, []byte("abcd"), got)
}
