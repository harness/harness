// Source: https://github.com/distribution/distribution

// Copyright 2014 https://github.com/distribution/distribution Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package base provides a base implementation of the storage driver that can
// be used to implement common checks. The goal is to increase the amount of
// code sharing.
//
// The canonical approach to use this class is to embed in the exported driver
// struct such that calls are proxied through this implementation. First,
// declare the internal driver, as follows:
//
//	type driver struct { ... internal ...}
//
// The resulting type should implement StorageDriver such that it can be the
// target of a Base struct. The exported type can then be declared as follows:
//
//	type Driver struct {
//		Base
//	}
//
// Because Driver embeds Base, it effectively implements Base. If the driver
// needs to intercept a call, before going to base, Driver should implement
// that method. Effectively, Driver can intercept calls before coming in and
// driver implements the actual logic.
//
// To further shield the embed from other packages, it is recommended to
// employ a private embed struct:
//
//	type baseEmbed struct {
//		base.Base
//	}
//
// Then, declare driver to embed baseEmbed, rather than Base directly:
//
//	type Driver struct {
//		baseEmbed
//	}
//
// The type now implements StorageDriver, proxying through Base, without
// exporting an unnecessary field.
package base

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/harness/gitness/registry/app/dist_temp/dcontext"
	"github.com/harness/gitness/registry/app/driver"
	"github.com/harness/gitness/registry/types"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog/log"
)

var (
	ObjectStoreOperationTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "object_store_operation_total",
		Help: "Total number of object store operations",
	}, []string{"operation", "provider", "bucket_id", "result", "error_type"})

	ObjectStoreOperationDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "object_store_operation_duration_seconds",
		Help: "Duration of object store operations in seconds",
		Buckets: []float64{
			0.001, 0.005, 0.01, 0.02, 0.03, 0.05, 0.075, 0.1, 0.15, 0.2, 0.3, 0.5, 1, 2, 5, 10, 30,
		},
		NativeHistogramBucketFactor:    1.1,
		NativeHistogramMaxBucketNumber: 100,
	}, []string{"operation", "provider", "bucket_id", "result", "error_type"})

	ObjectStoreBytesTransferredTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "object_store_bytes_transferred_total",
		Help: "Total bytes transferred in object store operations (get/put)",
	}, []string{"operation", "provider", "bucket_id"})
)

// Base provides a wrapper around a storagedriver implementation that provides
// common path and bounds checking.
type Base struct {
	driver.StorageDriver
	provider   string
	bucketID   string
	labelsOnce sync.Once
}

// SetMetricLabels configures the provider, bucket key and bucket ID labels
// used for Prometheus metrics. Safe to call from any goroutine;
// only the first call takes effect. The bucket_id label is stored as
// "bucketKey:bucketID" (e.g. "my-bucket:12").
func (base *Base) SetMetricLabels(provider string, bucketKey types.BucketKey, bucketID int64) {
	base.labelsOnce.Do(func() {
		base.provider = provider
		base.bucketID = fmt.Sprintf("%s:%d", bucketKey, bucketID)
	})
}

func (base *Base) metricProvider() string {
	if base.provider != "" {
		return base.provider
	}
	return base.Name()
}

func (base *Base) recordMetrics(
	operation string, start time.Time, err error, bytesTransferred int64,
) {
	duration := time.Since(start).Seconds()
	result := "success"
	errorType := ""
	if err != nil {
		errorType = classifyError(err)
		if operation == "stat" && errorType == "not_found" {
			errorType = ""
		} else {
			result = "error"
		}
	}

	provider := base.metricProvider()

	ObjectStoreOperationTotal.WithLabelValues(
		operation, provider, base.bucketID, result, errorType,
	).Inc()

	ObjectStoreOperationDurationSeconds.WithLabelValues(
		operation, provider, base.bucketID, result, errorType,
	).Observe(duration)

	if bytesTransferred > 0 && (operation == "get" || operation == "put") {
		ObjectStoreBytesTransferredTotal.WithLabelValues(
			operation, provider, base.bucketID,
		).Add(float64(bytesTransferred))
	}
}

func classifyError(err error) string {
	if err == nil {
		return ""
	}

	var pnfe driver.PathNotFoundError
	if errors.As(err, &pnfe) {
		return "not_found"
	}

	errStr := strings.ToLower(err.Error())
	if containsAny(errStr, "permission", "forbidden", "unauthorized", "access denied") {
		return "permission"
	}
	if containsAny(errStr, "timeout", "deadline", "context deadline exceeded") {
		return "timeout"
	}

	return "unknown"
}

func containsAny(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

// Format errors received from the storage driver.
func (base *Base) setDriverName(e error) error {
	if e == nil {
		return nil
	}
	switch {
	case errors.As(e, &driver.UnsupportedMethodError{}):
		var e1 driver.UnsupportedMethodError
		errors.As(e, &e1)
		e1.DriverName = base.StorageDriver.Name()
		return e1
	case errors.As(e, &driver.PathNotFoundError{}):
		var e2 driver.PathNotFoundError
		errors.As(e, &e2)
		e2.DriverName = base.StorageDriver.Name()
		return e2
	case errors.As(e, &driver.InvalidPathError{}):
		var e3 driver.InvalidPathError
		errors.As(e, &e3)
		e3.DriverName = base.StorageDriver.Name()
		return e3
	case errors.As(e, &driver.InvalidOffsetError{}):
		var e4 driver.InvalidOffsetError
		errors.As(e, &e4)
		e4.DriverName = base.StorageDriver.Name()
		return e4
	default:
		return driver.Error{
			DriverName: base.StorageDriver.Name(),
			Detail:     e,
		}
	}
}

// GetContent wraps GetContent of underlying storage driver.
func (base *Base) GetContent(ctx context.Context, path string) ([]byte, error) {
	ctx, done := dcontext.WithTrace(ctx)
	defer done("%s.GetContent(%q)", base.Name(), path)

	if !driver.PathRegexp.MatchString(path) {
		return nil, driver.InvalidPathError{Path: path, DriverName: base.StorageDriver.Name()}
	}

	start := time.Now()
	b, e := base.StorageDriver.GetContent(ctx, path)
	base.recordMetrics("get", start, e, int64(len(b)))
	return b, base.setDriverName(e)
}

// PutContent wraps PutContent of underlying storage driver.
func (base *Base) PutContent(ctx context.Context, path string, content []byte) error {
	ctx, done := dcontext.WithTrace(ctx)
	defer done("%s.PutContent(%q)", base.Name(), path)

	if !driver.PathRegexp.MatchString(path) {
		return driver.InvalidPathError{Path: path, DriverName: base.StorageDriver.Name()}
	}

	start := time.Now()
	err := base.StorageDriver.PutContent(ctx, path, content)
	base.recordMetrics("put", start, err, int64(len(content)))
	return base.setDriverName(err)
}

// Reader wraps Reader of underlying storage driver.
func (base *Base) Reader(ctx context.Context, path string, offset int64) (io.ReadCloser, error) {
	ctx, done := dcontext.WithTrace(ctx)
	defer done("%s.Reader(%q, %d)", base.Name(), path, offset)

	if offset < 0 {
		return nil, driver.InvalidOffsetError{Path: path, Offset: offset, DriverName: base.StorageDriver.Name()}
	}

	if !driver.PathRegexp.MatchString(path) {
		return nil, driver.InvalidPathError{Path: path, DriverName: base.StorageDriver.Name()}
	}

	start := time.Now()
	rc, e := base.StorageDriver.Reader(ctx, path, offset)
	if e != nil {
		base.recordMetrics("get", start, e, 0)
		return nil, base.setDriverName(e)
	}
	return &instrumentedReadCloser{
		ReadCloser: rc,
		base:       base,
		start:      start,
	}, nil
}

// Writer wraps Writer of underlying storage driver.
func (base *Base) Writer(ctx context.Context, path string, a bool) (driver.FileWriter, error) {
	ctx, done := dcontext.WithTrace(ctx)
	defer done("%s.Writer(%q, %v)", base.Name(), path, a)

	if !driver.PathRegexp.MatchString(path) {
		return nil, driver.InvalidPathError{Path: path, DriverName: base.StorageDriver.Name()}
	}

	start := time.Now()
	writer, e := base.StorageDriver.Writer(ctx, path, a)
	if e != nil {
		base.recordMetrics("put", start, e, 0)
		return nil, base.setDriverName(e)
	}
	return &instrumentedFileWriter{
		FileWriter: writer,
		base:       base,
		start:      start,
	}, nil
}

// Stat wraps Stat of underlying storage driver.
func (base *Base) Stat(ctx context.Context, path string) (driver.FileInfo, error) {
	ctx, done := dcontext.WithTrace(ctx)
	defer done("%s.Stat(%q)", base.Name(), path)

	if !driver.PathRegexp.MatchString(path) && path != "/" {
		return nil, driver.InvalidPathError{Path: path, DriverName: base.StorageDriver.Name()}
	}

	start := time.Now()
	fi, e := base.StorageDriver.Stat(ctx, path)
	base.recordMetrics("stat", start, e, 0)
	return fi, base.setDriverName(e)
}

// List wraps List of underlying storage driver.
func (base *Base) List(ctx context.Context, path string) ([]string, error) {
	ctx, done := dcontext.WithTrace(ctx)
	defer done("%s.List(%q)", base.Name(), path)

	if !driver.PathRegexp.MatchString(path) && path != "/" {
		return nil, driver.InvalidPathError{Path: path, DriverName: base.StorageDriver.Name()}
	}

	start := time.Now()
	str, e := base.StorageDriver.List(ctx, path)
	base.recordMetrics("list", start, e, 0)
	return str, base.setDriverName(e)
}

// Move wraps Move of underlying storage driver.
func (base *Base) Move(ctx context.Context, sourcePath string, destPath string) error {
	ctx, done := dcontext.WithTrace(ctx)
	defer done("%s.Move(%q, %q)", base.Name(), sourcePath, destPath)

	if !driver.PathRegexp.MatchString(sourcePath) {
		return driver.InvalidPathError{Path: sourcePath, DriverName: base.StorageDriver.Name()}
	} else if !driver.PathRegexp.MatchString(destPath) {
		return driver.InvalidPathError{Path: destPath, DriverName: base.StorageDriver.Name()}
	}

	start := time.Now()
	err := base.StorageDriver.Move(ctx, sourcePath, destPath)
	base.recordMetrics("move", start, err, 0)
	return base.setDriverName(err)
}

// Delete wraps Delete of underlying storage driver.
func (base *Base) Delete(ctx context.Context, path string) error {
	ctx, done := dcontext.WithTrace(ctx)
	defer done("%s.Delete(%q)", base.Name(), path)

	if !driver.PathRegexp.MatchString(path) {
		return driver.InvalidPathError{Path: path, DriverName: base.StorageDriver.Name()}
	}

	start := time.Now()
	err := base.StorageDriver.Delete(ctx, path)
	base.recordMetrics("delete", start, err, 0)
	return base.setDriverName(err)
}

// RedirectURL wraps RedirectURL of the underlying storage driver.
func (base *Base) RedirectURL(ctx context.Context, method string, path string, filename string) (string, error) {
	log.Ctx(ctx).Debug().Msgf("RedirectURL(%q, %q)", method, path)
	if !driver.PathRegexp.MatchString(path) {
		return "", driver.InvalidPathError{Path: path, DriverName: base.StorageDriver.Name()}
	}

	start := time.Now()
	str, e := base.StorageDriver.RedirectURL(ctx, method, path, filename)
	base.recordMetrics("redirect_url", start, e, 0)
	if str != "" {
		log.Ctx(ctx).Debug().Msg("redirect URL generated")
	}
	return str, base.setDriverName(e)
}

// Walk wraps Walk of underlying storage driver.
func (base *Base) Walk(ctx context.Context, path string, f driver.WalkFn, options ...func(*driver.WalkOptions)) error {
	ctx, done := dcontext.WithTrace(ctx)
	defer done("%s.Walk(%q)", base.Name(), path)

	if !driver.PathRegexp.MatchString(path) && path != "/" {
		return driver.InvalidPathError{Path: path, DriverName: base.StorageDriver.Name()}
	}

	start := time.Now()
	err := base.StorageDriver.Walk(ctx, path, f, options...)
	base.recordMetrics("walk", start, err, 0)
	return base.setDriverName(err)
}

// CopyObject wraps CopyObject of underlying storage driver.
func (base *Base) CopyObject(ctx context.Context, srcKey, destBucket, destKey string) error {
	start := time.Now()
	err := base.StorageDriver.CopyObject(ctx, srcKey, destBucket, destKey)
	base.recordMetrics("copy", start, err, 0)
	return base.setDriverName(err)
}

type instrumentedReadCloser struct {
	io.ReadCloser
	base          *Base
	start         time.Time
	bytesRead     int64
	readErr       error
	metricsCalled bool
}

func (r *instrumentedReadCloser) Read(p []byte) (n int, err error) {
	n, err = r.ReadCloser.Read(p)
	r.bytesRead += int64(n)
	if err != nil && err != io.EOF {
		r.readErr = err
	}
	return n, err
}

func (r *instrumentedReadCloser) Close() error {
	closErr := r.ReadCloser.Close()
	if !r.metricsCalled {
		r.base.recordMetrics("get", r.start, r.readErr, r.bytesRead)
		r.metricsCalled = true
	}
	return closErr
}

type instrumentedFileWriter struct {
	driver.FileWriter
	base          *Base
	start         time.Time
	metricsCalled bool
}

func (w *instrumentedFileWriter) Commit(ctx context.Context) error {
	err := w.FileWriter.Commit(ctx)
	if !w.metricsCalled {
		w.base.recordMetrics("put", w.start, err, w.FileWriter.Size())
		w.metricsCalled = true
	}
	return err
}

func (w *instrumentedFileWriter) Cancel(ctx context.Context) error {
	err := w.FileWriter.Cancel(ctx)
	if !w.metricsCalled {
		w.base.recordMetrics("put", w.start, err, 0)
		w.metricsCalled = true
	}
	return err
}

func (w *instrumentedFileWriter) Close() error {
	err := w.FileWriter.Close()
	if !w.metricsCalled {
		w.base.recordMetrics("put", w.start, err, w.FileWriter.Size())
		w.metricsCalled = true
	}
	return err
}
