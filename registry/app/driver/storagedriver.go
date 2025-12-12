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

package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// Version is a string representing the storage driver version, of the form
// Major.Minor.
// The registry must accept storage drivers with equal major version and greater
// minor version, but may not be compatible with older storage driver versions.
type Version string

// Major returns the major (primary) component of a version.
func (version Version) Major() uint {
	majorPart, _, _ := strings.Cut(string(version), ".")
	major, _ := strconv.ParseUint(majorPart, 10, 0)
	return uint(major)
}

// Minor returns the minor (secondary) component of a version.
func (version Version) Minor() uint {
	_, minorPart, _ := strings.Cut(string(version), ".")
	minor, _ := strconv.ParseUint(minorPart, 10, 0)
	return uint(minor)
}

// CurrentVersion is the current storage driver Version.
const CurrentVersion Version = "0.1"

// WalkOptions provides options to the walk function that may adjust its behaviour.
type WalkOptions struct {
	// If StartAfterHint is set, the walk may start with the first item lexographically
	// after the hint, but it is not guaranteed and drivers may start the walk from the path.
	StartAfterHint string
}

func WithStartAfterHint(startAfterHint string) func(*WalkOptions) {
	return func(s *WalkOptions) {
		s.StartAfterHint = startAfterHint
	}
}

// StorageDriver defines methods that a Storage Driver must implement for a
// filesystem-like key/value object storage. Storage Drivers are automatically
// registered via an internal registration mechanism, and generally created
// via the StorageDriverFactory interface
// (https://godoc.org/github.com/distribution/distribution/registry/storage/driver/factory).
// Please see the aforementioned factory package for example code showing how to get an instance
// of a StorageDriver.
type StorageDriver interface {
	StorageDeleter

	// Name returns the human-readable "name" of the driver, useful in error
	// messages and logging. By convention, this will just be the registration
	// name, but drivers may provide other information here.
	Name() string

	// GetContent retrieves the content stored at "path" as a []byte.
	// This should primarily be used for small objects.
	GetContent(ctx context.Context, path string) ([]byte, error)

	// PutContent stores the []byte content at a location designated by "path".
	// This should primarily be used for small objects.
	PutContent(ctx context.Context, path string, content []byte) error

	// Reader retrieves an io.ReadCloser for the content stored at "path"
	// with a given byte offset.
	// May be used to resume reading a stream by providing a nonzero offset.
	Reader(ctx context.Context, path string, offset int64) (io.ReadCloser, error)

	// Writer returns a FileWriter which will store the content written to it
	// at the location designated by "path" after the call to Commit.
	// A path may be appended to if it has not been committed, or if the
	// existing committed content is zero length.
	//
	// The behaviour of appending to paths with non-empty committed content is
	// undefined. Specific implementations may document their own behavior.
	Writer(ctx context.Context, path string, a bool) (FileWriter, error)

	// Stat retrieves the FileInfo for the given path, including the current
	// size in bytes and the creation time.
	Stat(ctx context.Context, path string) (FileInfo, error)

	// List returns a list of the objects that are direct descendants of the
	// given path.
	List(ctx context.Context, path string) ([]string, error)

	// Move moves an object stored at sourcePath to destPath, removing the
	// original object.
	Move(ctx context.Context, sourcePath string, destPath string) error

	// RedirectURL returns a URL which the client of the request r may use
	// to retrieve the content stored at path. Returning the empty string
	// signals that the request may not be redirected.
	RedirectURL(ctx context.Context, method string, path string, filename string) (string, error)

	// Walk traverses a filesystem defined within driver, starting
	// from the given path, calling f on each file.
	// If the returned error from the WalkFn is ErrSkipDir and fileInfo refers
	// to a directory, the directory will not be entered and Walk
	// will continue the traversal.
	// If the returned error from the WalkFn is ErrFilledBuffer, processing stops.
	Walk(ctx context.Context, path string, f WalkFn, options ...func(*WalkOptions)) error

	// CopyObject performs a server-side copy of an object from one location to another.
	// For S3-compatible storage, this uses the efficient CopyObject API call instead of download-then-upload.
	CopyObject(ctx context.Context, srcKey, destBucket, destKey string) error
}

// StorageDeleter defines methods that a Storage Driver must implement to delete objects.
// This allows using a narrower interface than StorageDriver when we only need the delete functionality, such as when
// mocking a storage driver for testing online garbage collection.
type StorageDeleter interface {
	// Delete recursively deletes all objects stored at "path" and its subpaths.
	Delete(ctx context.Context, path string) error
}

// FileWriter provides an abstraction for an opened writable file-like object in
// the storage backend. The FileWriter must flush all content written to it on
// the call to Close, but is only required to make its content readable on a
// call to Commit.
type FileWriter interface {
	io.WriteCloser

	// Size returns the number of bytes written to this FileWriter.
	Size() int64

	// Cancel removes any written content from this FileWriter.
	Cancel(context.Context) error

	// Commit flushes all content written to this FileWriter and makes it
	// available for future calls to StorageDriver.GetContent and
	// StorageDriver.Reader.
	Commit(context.Context) error
}

// PathRegexp is the regular expression which each file path must match. A
// file path is absolute, beginning with a slash and containing a positive
// number of path components separated by slashes, where each component is
// restricted to alphanumeric characters or a period, underscore, or
// hyphen.
var PathRegexp = regexp.MustCompile(`^(/[A-Za-z0-9._-]+)+$`)

// UnsupportedMethodError may be returned in the case where a
// StorageDriver implementation does not support an optional method.
type UnsupportedMethodError struct {
	DriverName string
}

func (err UnsupportedMethodError) Error() string {
	return fmt.Sprintf("%s: unsupported method", err.DriverName)
}

// PathNotFoundError is returned when operating on a nonexistent path.
type PathNotFoundError struct {
	Path       string
	DriverName string
}

func (err PathNotFoundError) Error() string {
	return fmt.Sprintf("%s: Path not found: %s", err.DriverName, err.Path)
}

// InvalidPathError is returned when the provided path is malformed.
type InvalidPathError struct {
	Path       string
	DriverName string
}

func (err InvalidPathError) Error() string {
	return fmt.Sprintf("%s: invalid path: %s", err.DriverName, err.Path)
}

// InvalidOffsetError is returned when attempting to read or write from an
// invalid offset.
type InvalidOffsetError struct {
	Path       string
	Offset     int64
	DriverName string
}

func (err InvalidOffsetError) Error() string {
	return fmt.Sprintf("%s: invalid offset: %d for path: %s", err.DriverName, err.Offset, err.Path)
}

// Error is a catch-all error type which captures an error string and
// the driver type on which it occurred.
type Error struct {
	DriverName string
	Detail     error
}

func (err Error) Error() string {
	return fmt.Sprintf("%s: %s", err.DriverName, err.Detail)
}

func (err Error) MarshalJSON() ([]byte, error) {
	return json.Marshal(
		struct {
			DriverName string `json:"driver"`
			Detail     string `json:"detail"`
		}{
			DriverName: err.DriverName,
			Detail:     err.Detail.Error(),
		},
	)
}

// StorageDriverError provides the envelope for multiple errors
// for use within the storagedriver implementations.
type StorageDriverError struct {
	DriverName string
	Errs       []error
}

var _ error = StorageDriverError{}

func (e StorageDriverError) Error() string {
	switch len(e.Errs) {
	case 0:
		return fmt.Sprintf("%s: <nil>", e.DriverName)
	case 1:
		return fmt.Sprintf("%s: %s", e.DriverName, e.Errs[0].Error())
	default:
		msg := "errors:\n"
		for _, err := range e.Errs {
			msg += err.Error() + "\n"
		}
		return fmt.Sprintf("%s: %s", e.DriverName, msg)
	}
}

// MarshalJSON converts slice of errors into the format
// that is serializable by JSON.
func (e StorageDriverError) MarshalJSON() ([]byte, error) {
	tmpErrs := struct {
		DriverName string   `json:"driver"`
		Details    []string `json:"details"`
	}{
		DriverName: e.DriverName,
	}

	if len(e.Errs) == 0 {
		tmpErrs.Details = make([]string, 0)
		return json.Marshal(tmpErrs)
	}

	for _, err := range e.Errs {
		tmpErrs.Details = append(tmpErrs.Details, err.Error())
	}

	return json.Marshal(tmpErrs)
}
