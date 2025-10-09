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
	"io"

	"github.com/harness/gitness/registry/app/dist_temp/dcontext"
	"github.com/harness/gitness/registry/app/driver"

	"github.com/rs/zerolog/log"
)

func init() {

}

// Base provides a wrapper around a storagedriver implementation that provides
// common path and bounds checking.
type Base struct {
	driver.StorageDriver
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

	b, e := base.StorageDriver.GetContent(ctx, path)
	return b, base.setDriverName(e)
}

// PutContent wraps PutContent of underlying storage driver.
func (base *Base) PutContent(ctx context.Context, path string, content []byte) error {
	ctx, done := dcontext.WithTrace(ctx)
	defer done("%s.PutContent(%q)", base.Name(), path)

	if !driver.PathRegexp.MatchString(path) {
		return driver.InvalidPathError{Path: path, DriverName: base.StorageDriver.Name()}
	}

	err := base.setDriverName(base.StorageDriver.PutContent(ctx, path, content))
	return err
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

	rc, e := base.StorageDriver.Reader(ctx, path, offset)
	return rc, base.setDriverName(e)
}

// Writer wraps Writer of underlying storage driver.
func (base *Base) Writer(ctx context.Context, path string, a bool) (driver.FileWriter, error) {
	ctx, done := dcontext.WithTrace(ctx)
	defer done("%s.Writer(%q, %v)", base.Name(), path, a)

	if !driver.PathRegexp.MatchString(path) {
		return nil, driver.InvalidPathError{Path: path, DriverName: base.StorageDriver.Name()}
	}

	writer, e := base.StorageDriver.Writer(ctx, path, a)
	return writer, base.setDriverName(e)
}

// Stat wraps Stat of underlying storage driver.
func (base *Base) Stat(ctx context.Context, path string) (driver.FileInfo, error) {
	ctx, done := dcontext.WithTrace(ctx)
	defer done("%s.Stat(%q)", base.Name(), path)

	if !driver.PathRegexp.MatchString(path) && path != "/" {
		return nil, driver.InvalidPathError{Path: path, DriverName: base.StorageDriver.Name()}
	}

	fi, e := base.StorageDriver.Stat(ctx, path)
	return fi, base.setDriverName(e)
}

// List wraps List of underlying storage driver.
func (base *Base) List(ctx context.Context, path string) ([]string, error) {
	ctx, done := dcontext.WithTrace(ctx)
	defer done("%s.List(%q)", base.Name(), path)

	if !driver.PathRegexp.MatchString(path) && path != "/" {
		return nil, driver.InvalidPathError{Path: path, DriverName: base.StorageDriver.Name()}
	}

	str, e := base.StorageDriver.List(ctx, path)
	return str, base.setDriverName(e)
}

// Move wraps Move of underlying storage driver.
func (base *Base) Move(ctx context.Context, sourcePath string, destPath string) error {
	ctx, done := dcontext.WithTrace(ctx)
	defer done("%s.Move(%q, %q", base.Name(), sourcePath, destPath)

	if !driver.PathRegexp.MatchString(sourcePath) {
		return driver.InvalidPathError{Path: sourcePath, DriverName: base.StorageDriver.Name()}
	} else if !driver.PathRegexp.MatchString(destPath) {
		return driver.InvalidPathError{Path: destPath, DriverName: base.StorageDriver.Name()}
	}

	err := base.setDriverName(base.StorageDriver.Move(ctx, sourcePath, destPath))
	return err
}

// Delete wraps Delete of underlying storage driver.
func (base *Base) Delete(ctx context.Context, path string) error {
	ctx, done := dcontext.WithTrace(ctx)
	defer done("%s.Delete(%q)", base.Name(), path)

	if !driver.PathRegexp.MatchString(path) {
		return driver.InvalidPathError{Path: path, DriverName: base.StorageDriver.Name()}
	}

	err := base.setDriverName(base.StorageDriver.Delete(ctx, path))
	return err
}

// RedirectURL wraps RedirectURL of the underlying storage driver.
func (base *Base) RedirectURL(ctx context.Context, method string, path string, filename string) (string, error) {
	log.Ctx(ctx).Info().Msgf("RedirectURL(%q, %q)", method, path)
	if !driver.PathRegexp.MatchString(path) {
		return "", driver.InvalidPathError{Path: path, DriverName: base.StorageDriver.Name()}
	}

	str, e := base.StorageDriver.RedirectURL(ctx, method, path, filename)
	log.Ctx(ctx).Info().Msgf("Redirect URL generated %s", str)
	return str, base.setDriverName(e)
}

// Walk wraps Walk of underlying storage driver.
func (base *Base) Walk(ctx context.Context, path string, f driver.WalkFn, options ...func(*driver.WalkOptions)) error {
	ctx, done := dcontext.WithTrace(ctx)
	defer done("%s.Walk(%q)", base.Name(), path)

	if !driver.PathRegexp.MatchString(path) && path != "/" {
		return driver.InvalidPathError{Path: path, DriverName: base.StorageDriver.Name()}
	}

	return base.setDriverName(base.StorageDriver.Walk(ctx, path, f, options...))
}
