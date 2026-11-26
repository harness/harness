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

package base

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/harness/gitness/registry/app/driver"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

var errRb5Boom = errors.New("rb5 boom")

const rb5Path = "/rb5/path"

// rb5Counter reads a labelled counter from the default registry.
func rb5Counter(t *testing.T, name string, labels map[string]string) float64 {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.GetMetric() {
			matched := 0
			for _, label := range metric.GetLabel() {
				if want, ok := labels[label.GetName()]; ok && want == label.GetValue() {
					matched++
				}
			}
			if matched == len(labels) {
				return metric.GetCounter().GetValue()
			}
		}
	}
	return 0
}

func rb5Ops(t *testing.T, op, provider, bucket, result, errType string) float64 {
	t.Helper()
	return rb5Counter(t, "object_store_operation_total", map[string]string{
		"operation": op, "provider": provider, "bucket_id": bucket, "result": result, "error_type": errType,
	})
}

// rb5Inner is a configurable inner driver; every method fails with err when it is set.
type rb5Inner struct {
	driver.StorageDriver
	err       error
	readErr   error
	closeErr  error
	commitErr error
	cancelErr error
}

func (d *rb5Inner) Name() string { return "rb5inner" }

func (d *rb5Inner) GetContent(context.Context, string) ([]byte, error) {
	return []byte("data"), d.err
}

func (d *rb5Inner) PutContent(context.Context, string, []byte) error { return d.err }

func (d *rb5Inner) Reader(context.Context, string, int64) (io.ReadCloser, error) {
	if d.err != nil {
		return nil, d.err
	}
	return &rb5ReadCloser{err: d.readErr, closeErr: d.closeErr}, nil
}

func (d *rb5Inner) Writer(context.Context, string, bool) (driver.FileWriter, error) {
	if d.err != nil {
		return nil, d.err
	}
	return &rb5FileWriter{commitErr: d.commitErr, closeErr: d.closeErr, cancelErr: d.cancelErr}, nil
}

func (d *rb5Inner) Stat(context.Context, string) (driver.FileInfo, error) {
	return driver.FileInfoInternal{FileInfoFields: driver.FileInfoFields{Path: rb5Path, Size: 4}}, d.err
}

func (d *rb5Inner) List(context.Context, string) ([]string, error) { return []string{"a"}, d.err }

func (d *rb5Inner) Move(context.Context, string, string) error { return d.err }

func (d *rb5Inner) Delete(context.Context, string) error { return d.err }

func (d *rb5Inner) RedirectURL(context.Context, string, string, string) (string, error) {
	return "https://example.com/x", d.err
}

func (d *rb5Inner) Walk(context.Context, string, driver.WalkFn, ...func(*driver.WalkOptions)) error {
	return d.err
}

func (d *rb5Inner) CopyObject(context.Context, string, string, string) error { return d.err }

type rb5ReadCloser struct {
	err      error
	closeErr error
	done     bool
}

func (r *rb5ReadCloser) Read(p []byte) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	if r.done {
		return 0, io.EOF
	}
	r.done = true
	return copy(p, "data"), nil
}

func (r *rb5ReadCloser) Close() error { return r.closeErr }

type rb5FileWriter struct {
	commitErr error
	closeErr  error
	cancelErr error
}

func (w *rb5FileWriter) Write(p []byte) (int, error) { return len(p), nil }
func (w *rb5FileWriter) Close() error                { return w.closeErr }
func (w *rb5FileWriter) Size() int64                 { return 4 }
func (w *rb5FileWriter) Cancel(context.Context) error {
	return w.cancelErr
}
func (w *rb5FileWriter) Commit(context.Context) error { return w.commitErr }

func TestRb5ClassifyError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, ""},
		{"not found", driver.PathNotFoundError{Path: "/x"}, "not_found"},
		{"wrapped not found", errors.Join(errRb5Boom, driver.PathNotFoundError{Path: "/x"}), "not_found"},
		{"permission", errors.New("Access Denied by policy"), "permission"},
		{"forbidden", errors.New("403 forbidden"), "permission"},
		{"unauthorized", errors.New("unauthorized"), "permission"},
		{"timeout", errors.New("context deadline exceeded"), "timeout"},
		{"timeout word", errors.New("request timeout"), "timeout"},
		{"unknown", errRb5Boom, "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, classifyError(tt.err))
		})
	}
	require.True(t, containsAny("hello world", "xyz", "world"))
	require.False(t, containsAny("hello", "xyz"))
	require.False(t, containsAny("hello"))
}

func TestRb5SetDriverName(t *testing.T) {
	b := &Base{StorageDriver: &rb5Inner{}}

	require.NoError(t, b.setDriverName(nil))

	var unsupported driver.UnsupportedMethodError
	require.ErrorAs(t, b.setDriverName(driver.UnsupportedMethodError{}), &unsupported)
	require.Equal(t, "rb5inner", unsupported.DriverName)

	var notFound driver.PathNotFoundError
	require.ErrorAs(t, b.setDriverName(driver.PathNotFoundError{Path: "/p"}), &notFound)
	require.Equal(t, "rb5inner", notFound.DriverName)
	require.Equal(t, "/p", notFound.Path)

	var invalidPath driver.InvalidPathError
	require.ErrorAs(t, b.setDriverName(driver.InvalidPathError{Path: "/p"}), &invalidPath)
	require.Equal(t, "rb5inner", invalidPath.DriverName)

	var invalidOffset driver.InvalidOffsetError
	require.ErrorAs(t, b.setDriverName(driver.InvalidOffsetError{Path: "/p", Offset: -1}), &invalidOffset)
	require.Equal(t, "rb5inner", invalidOffset.DriverName)
	require.Equal(t, int64(-1), invalidOffset.Offset)

	var wrapped driver.Error
	require.ErrorAs(t, b.setDriverName(errRb5Boom), &wrapped)
	require.Equal(t, "rb5inner", wrapped.DriverName)
	require.ErrorIs(t, wrapped.Detail, errRb5Boom)
}

func TestRb5BaseInvalidPaths(t *testing.T) {
	ctx := context.Background()
	b := &Base{StorageDriver: &rb5Inner{}}
	const bad = "no-leading-slash"

	var invalidPath driver.InvalidPathError
	check := func(err error) {
		t.Helper()
		require.ErrorAs(t, err, &invalidPath)
	}

	_, err := b.GetContent(ctx, bad)
	check(err)
	check(b.PutContent(ctx, bad, nil))
	_, err = b.Reader(ctx, bad, 0)
	check(err)
	_, err = b.Writer(ctx, bad, false)
	check(err)
	_, err = b.Stat(ctx, bad)
	check(err)
	_, err = b.List(ctx, bad)
	check(err)
	check(b.Move(ctx, bad, rb5Path))
	check(b.Move(ctx, rb5Path, bad))
	check(b.Delete(ctx, bad))
	_, err = b.RedirectURL(ctx, "GET", bad, "")
	check(err)
	check(b.Walk(ctx, bad, nil))

	_, err = b.Reader(ctx, rb5Path, -1)
	require.ErrorAs(t, err, &driver.InvalidOffsetError{})

	// The root path is accepted by the operations that work on directories.
	_, err = b.Stat(ctx, "/")
	require.NoError(t, err)
	_, err = b.List(ctx, "/")
	require.NoError(t, err)
	require.NoError(t, b.Walk(ctx, "/", nil))
}

func TestRb5BaseDriverErrors(t *testing.T) {
	ctx := context.Background()
	b := &Base{StorageDriver: &rb5Inner{err: errRb5Boom}}

	var wrapped driver.Error
	check := func(err error) {
		t.Helper()
		require.ErrorAs(t, err, &wrapped)
		require.Equal(t, "rb5inner", wrapped.DriverName)
	}

	_, err := b.GetContent(ctx, rb5Path)
	check(err)
	check(b.PutContent(ctx, rb5Path, []byte("x")))
	_, err = b.Reader(ctx, rb5Path, 0)
	check(err)
	_, err = b.Writer(ctx, rb5Path, false)
	check(err)
	_, err = b.Stat(ctx, rb5Path)
	check(err)
	_, err = b.List(ctx, rb5Path)
	check(err)
	check(b.Move(ctx, rb5Path, rb5Path))
	check(b.Delete(ctx, rb5Path))
	_, err = b.RedirectURL(ctx, "GET", rb5Path, "")
	check(err)
	check(b.Walk(ctx, rb5Path, nil))
	check(b.CopyObject(ctx, "a", "bucket", "b"))
}

func TestRb5BaseSuccessAndMetrics(t *testing.T) {
	ctx := context.Background()
	b := &Base{StorageDriver: &rb5Inner{}}
	b.SetMetricLabels("rb5provider", "rb5bucket", 7)
	// Only the first call takes effect.
	b.SetMetricLabels("other", "other", 9)

	count := func(op, result, errType string) float64 {
		return rb5Ops(t, op, "rb5provider", "rb5bucket:7", result, errType)
	}

	content, err := b.GetContent(ctx, rb5Path)
	require.NoError(t, err)
	require.Equal(t, "data", string(content))
	require.Equal(t, 1.0, count("get", "success", ""))
	bytesLabels := map[string]string{"operation": "get", "provider": "rb5provider"}
	require.Equal(t, 4.0, rb5Counter(t, "object_store_bytes_transferred_total", bytesLabels))

	require.NoError(t, b.PutContent(ctx, rb5Path, []byte("data")))
	require.Equal(t, 1.0, count("put", "success", ""))

	_, err = b.Stat(ctx, rb5Path)
	require.NoError(t, err)
	_, err = b.List(ctx, rb5Path)
	require.NoError(t, err)
	require.NoError(t, b.Move(ctx, rb5Path, rb5Path))
	require.NoError(t, b.Delete(ctx, rb5Path))
	require.NoError(t, b.Walk(ctx, rb5Path, nil))
	require.NoError(t, b.CopyObject(ctx, "a", "bucket", "b"))
	url, err := b.RedirectURL(ctx, "GET", rb5Path, "f")
	require.NoError(t, err)
	require.Equal(t, "https://example.com/x", url)
	for _, op := range []string{"stat", "list", "move", "delete", "walk", "copy", "redirect_url"} {
		require.Equal(t, 1.0, count(op, "success", ""), op)
	}

	// A stat on a missing path is not counted as an error.
	missing := &Base{StorageDriver: &rb5Inner{err: driver.PathNotFoundError{Path: rb5Path}}}
	missing.SetMetricLabels("rb5missing", "rb5bucket", 1)
	_, err = missing.Stat(ctx, rb5Path)
	require.ErrorAs(t, err, &driver.PathNotFoundError{})
	require.Equal(t, 1.0,
		rb5Ops(t, "stat", "rb5missing", "rb5bucket:1", "success", ""))
	_, err = missing.GetContent(ctx, rb5Path)
	require.Error(t, err)
	require.Equal(t, 1.0,
		rb5Ops(t, "get", "rb5missing", "rb5bucket:1", "error", "not_found"))

	// Without labels the driver name is the provider.
	require.Equal(t, "rb5inner", (&Base{StorageDriver: &rb5Inner{}}).metricProvider())
}

func TestRb5InstrumentedReader(t *testing.T) {
	ctx := context.Background()

	b := &Base{StorageDriver: &rb5Inner{}}
	b.SetMetricLabels("rb5reader", "rb5bucket", 2)
	rc, err := b.Reader(ctx, rb5Path, 0)
	require.NoError(t, err)
	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Equal(t, "data", string(data))
	require.NoError(t, rc.Close())
	require.NoError(t, rc.Close())
	require.Equal(t, 1.0,
		rb5Ops(t, "get", "rb5reader", "rb5bucket:2", "success", ""))

	failing := &Base{StorageDriver: &rb5Inner{readErr: errRb5Boom, closeErr: errRb5Boom}}
	failing.SetMetricLabels("rb5readfail", "rb5bucket", 3)
	rc, err = failing.Reader(ctx, rb5Path, 0)
	require.NoError(t, err)
	_, err = rc.Read(make([]byte, 4))
	require.ErrorIs(t, err, errRb5Boom)
	require.ErrorIs(t, rc.Close(), errRb5Boom)
	require.Equal(t, 1.0,
		rb5Ops(t, "get", "rb5readfail", "rb5bucket:3", "error", "unknown"))
}

func TestRb5InstrumentedWriter(t *testing.T) {
	ctx := context.Background()
	opCount := func(provider, bucket, result, errType string) float64 {
		return rb5Ops(t, "put", provider, bucket, result, errType)
	}

	tests := []struct {
		name     string
		inner    *rb5Inner
		finish   func(w driver.FileWriter) error
		wantErr  bool
		resultFn func(provider, bucket string) float64
	}{
		{
			name:   "commit",
			inner:  &rb5Inner{},
			finish: func(w driver.FileWriter) error { return w.Commit(ctx) },
			resultFn: func(provider, bucket string) float64 {
				return opCount(provider, bucket, "success", "")
			},
		},
		{
			name:    "commit error",
			inner:   &rb5Inner{commitErr: errRb5Boom},
			finish:  func(w driver.FileWriter) error { return w.Commit(ctx) },
			wantErr: true,
			resultFn: func(provider, bucket string) float64 {
				return opCount(provider, bucket, "error", "unknown")
			},
		},
		{
			name:   "cancel",
			inner:  &rb5Inner{},
			finish: func(w driver.FileWriter) error { return w.Cancel(ctx) },
			resultFn: func(provider, bucket string) float64 {
				return opCount(provider, bucket, "success", "")
			},
		},
		{
			name:    "cancel error",
			inner:   &rb5Inner{cancelErr: errRb5Boom},
			finish:  func(w driver.FileWriter) error { return w.Cancel(ctx) },
			wantErr: true,
			resultFn: func(provider, bucket string) float64 {
				return opCount(provider, bucket, "error", "unknown")
			},
		},
		{
			name:   "close",
			inner:  &rb5Inner{},
			finish: func(w driver.FileWriter) error { return w.Close() },
			resultFn: func(provider, bucket string) float64 {
				return opCount(provider, bucket, "success", "")
			},
		},
		{
			name:    "close error",
			inner:   &rb5Inner{closeErr: errRb5Boom},
			finish:  func(w driver.FileWriter) error { return w.Close() },
			wantErr: true,
			resultFn: func(provider, bucket string) float64 {
				return opCount(provider, bucket, "error", "unknown")
			},
		},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := "rb5writer" + strings.Repeat("x", i)
			b := &Base{StorageDriver: tt.inner}
			b.SetMetricLabels(provider, "rb5bucket", 4)
			w, err := b.Writer(ctx, rb5Path, false)
			require.NoError(t, err)
			_, err = w.Write([]byte("data"))
			require.NoError(t, err)

			err = tt.finish(w)
			if tt.wantErr {
				require.ErrorIs(t, err, errRb5Boom)
			} else {
				require.NoError(t, err)
			}
			// Metrics are recorded only once, even if the writer is closed afterwards.
			_ = w.Close()
			require.Equal(t, 1.0, tt.resultFn(provider, "rb5bucket:4"))
		})
	}
}

func TestRb5RegulatorLimitParameter(t *testing.T) {
	tests := []struct {
		name    string
		param   any
		want    uint64
		wantErr bool
	}{
		{"string", "12", 12, false},
		{"hex string", "0x10", 16, false},
		{"bad string", "abc", 0, true},
		{"uint64", uint64(7), 7, false},
		{"int", 9, 9, false},
		{"int32", int32(9), 9, false},
		{"int64", int64(9), 9, false},
		{"negative int", -4, 3, false},
		{"zero int64", int64(0), 3, false},
		{"uint", uint(11), 11, false},
		{"uint32", uint32(11), 11, false},
		{"nil uses default", nil, 5, false},
		{"below minimum", uint64(1), 3, false},
		{"invalid type", 1.5, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetLimitFromParameter(tt.param, 3, 5)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestRb5RegulatorDelegates(t *testing.T) {
	ctx := context.Background()
	inner := &rb5Inner{}
	r := NewRegulator(inner, 2)

	require.Equal(t, "rb5inner", r.Name())
	content, err := r.GetContent(ctx, rb5Path)
	require.NoError(t, err)
	require.Equal(t, "data", string(content))
	require.NoError(t, r.PutContent(ctx, rb5Path, nil))

	rc, err := r.Reader(ctx, rb5Path, 0)
	require.NoError(t, err)
	require.NoError(t, rc.Close())

	w, err := r.Writer(ctx, rb5Path, false)
	require.NoError(t, err)
	require.Equal(t, int64(4), w.Size())

	_, err = r.Stat(ctx, rb5Path)
	require.NoError(t, err)
	_, err = r.List(ctx, rb5Path)
	require.NoError(t, err)
	require.NoError(t, r.Move(ctx, rb5Path, rb5Path))
	require.NoError(t, r.Delete(ctx, rb5Path))
	_, err = r.RedirectURL(ctx, "GET", rb5Path, "")
	require.NoError(t, err)
	require.NoError(t, r.Walk(ctx, rb5Path, nil))
	require.NoError(t, r.CopyObject(ctx, "a", "b", "c"))

	inner.err = errRb5Boom
	_, err = r.Reader(ctx, rb5Path, 0)
	require.ErrorIs(t, err, errRb5Boom)
	_, err = r.Writer(ctx, rb5Path, false)
	require.ErrorIs(t, err, errRb5Boom)
}
