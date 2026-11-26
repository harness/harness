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

package base

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"testing"

	"github.com/harness/gitness/registry/app/driver"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const rsgBasePath = "/rsg/base/file"

type rsgMemDriver struct {
	mu   sync.Mutex
	data map[string][]byte
}

func rsgNewMemDriver() *rsgMemDriver {
	return &rsgMemDriver{data: make(map[string][]byte)}
}

func (m *rsgMemDriver) Name() string { return "rsg-mem" }

func (m *rsgMemDriver) GetContent(_ context.Context, path string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.data[path]
	if !ok {
		return nil, driver.PathNotFoundError{Path: path, DriverName: m.Name()}
	}
	return bytes.Clone(b), nil
}

func (m *rsgMemDriver) PutContent(_ context.Context, path string, content []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[path] = bytes.Clone(content)
	return nil
}

func (m *rsgMemDriver) Reader(ctx context.Context, path string, offset int64) (io.ReadCloser, error) {
	b, err := m.GetContent(ctx, path)
	if err != nil {
		return nil, err
	}
	if offset > int64(len(b)) {
		return nil, driver.InvalidOffsetError{Path: path, Offset: offset, DriverName: m.Name()}
	}
	return io.NopCloser(bytes.NewReader(b[offset:])), nil
}

func (m *rsgMemDriver) Writer(_ context.Context, path string, _ bool) (driver.FileWriter, error) {
	return &rsgMemWriter{driver: m, path: path}, nil
}

func (m *rsgMemDriver) Stat(_ context.Context, path string) (driver.FileInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.data[path]
	if !ok {
		return nil, driver.PathNotFoundError{Path: path, DriverName: m.Name()}
	}
	return driver.FileInfoInternal{
		FileInfoFields: driver.FileInfoFields{Path: path, Size: int64(len(b))},
	}, nil
}

func (m *rsgMemDriver) List(_ context.Context, path string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	prefix := path + "/"
	var out []string
	for p := range m.data {
		if len(path) == 1 && path == "/" {
			out = append(out, p)
			continue
		}
		if len(p) > len(prefix) && p[:len(prefix)] == prefix {
			out = append(out, p)
		}
	}
	return out, nil
}

func (m *rsgMemDriver) Move(_ context.Context, src, dst string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.data[src]
	if !ok {
		return driver.PathNotFoundError{Path: src, DriverName: m.Name()}
	}
	m.data[dst] = b
	delete(m.data, src)
	return nil
}

func (m *rsgMemDriver) Delete(_ context.Context, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[path]; !ok {
		return driver.PathNotFoundError{Path: path, DriverName: m.Name()}
	}
	delete(m.data, path)
	return nil
}

func (m *rsgMemDriver) RedirectURL(_ context.Context, _, _ string, _ string) (string, error) {
	return "", nil
}

func (m *rsgMemDriver) Walk(_ context.Context, _ string, _ driver.WalkFn, _ ...func(*driver.WalkOptions)) error {
	return nil
}

func (m *rsgMemDriver) CopyObject(ctx context.Context, srcKey, _, destKey string) error {
	return m.Move(ctx, srcKey, destKey)
}

type rsgMemWriter struct {
	driver  *rsgMemDriver
	path    string
	content []byte
	done    bool
}

func (w *rsgMemWriter) Write(p []byte) (int, error) {
	if w.done {
		return 0, errors.New("closed")
	}
	w.content = append(w.content, p...)
	return len(p), nil
}

func (w *rsgMemWriter) Close() error {
	w.done = true
	return nil
}

func (w *rsgMemWriter) Size() int64 { return int64(len(w.content)) }

func (w *rsgMemWriter) Cancel(_ context.Context) error {
	w.content = nil
	w.done = true
	return nil
}

func (w *rsgMemWriter) Commit(ctx context.Context) error {
	return w.driver.PutContent(ctx, w.path, w.content)
}

func TestRsg_BaseInvalidPath(t *testing.T) {
	ctx := context.Background()
	b := &Base{StorageDriver: rsgNewMemDriver()}

	_, err := b.GetContent(ctx, "bad path")
	require.Error(t, err)
	var ipe driver.InvalidPathError
	require.True(t, errors.As(err, &ipe))
}

func TestRsg_BaseInvalidOffset(t *testing.T) {
	ctx := context.Background()
	b := &Base{StorageDriver: rsgNewMemDriver()}
	_, err := b.Reader(ctx, rsgBasePath, -1)
	require.Error(t, err)
	var ioe driver.InvalidOffsetError
	require.True(t, errors.As(err, &ioe))
}

func TestRsg_BasePutGetStatList(t *testing.T) {
	ctx := context.Background()
	b := &Base{StorageDriver: rsgNewMemDriver()}
	payload := []byte("base payload")

	require.NoError(t, b.PutContent(ctx, rsgBasePath, payload))
	got, err := b.GetContent(ctx, rsgBasePath)
	require.NoError(t, err)
	assert.Equal(t, payload, got)

	fi, err := b.Stat(ctx, rsgBasePath)
	require.NoError(t, err)
	assert.Equal(t, int64(len(payload)), fi.Size())
}

func TestRsg_BaseWriterReaderMetrics(t *testing.T) {
	ctx := context.Background()
	b := &Base{StorageDriver: rsgNewMemDriver()}
	b.SetMetricLabels("mem", types.BucketKey("bk"), 42)

	w, err := b.Writer(ctx, "/rsg/metrics", false)
	require.NoError(t, err)
	_, err = w.Write([]byte("m"))
	require.NoError(t, err)
	require.NoError(t, w.Commit(ctx))
	require.NoError(t, w.Close())

	rc, err := b.Reader(ctx, "/rsg/metrics", 0)
	require.NoError(t, err)
	defer rc.Close()
	_, err = io.ReadAll(rc)
	require.NoError(t, err)
	require.NoError(t, rc.Close())
}

func TestRsg_BaseMoveDeleteWalkCopy(t *testing.T) {
	ctx := context.Background()
	b := &Base{StorageDriver: rsgNewMemDriver()}
	require.NoError(t, b.PutContent(ctx, "/rsg/mv", []byte("mv")))
	require.NoError(t, b.Move(ctx, "/rsg/mv", "/rsg/mv2"))
	require.NoError(t, b.CopyObject(ctx, "/rsg/mv2", "", "/rsg/mv3"))
	require.NoError(t, b.Delete(ctx, "/rsg/mv3"))

	err := b.Walk(ctx, "/rsg", func(fi driver.FileInfo) error { return nil })
	require.NoError(t, err)

	_, err = b.RedirectURL(ctx, "GET", "/rsg/mv2", "f")
	require.NoError(t, err)
}

func TestRsg_BaseNotFoundSetsDriverName(t *testing.T) {
	ctx := context.Background()
	b := &Base{StorageDriver: rsgNewMemDriver()}
	_, err := b.GetContent(ctx, "/rsg/missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rsg-mem")
}
