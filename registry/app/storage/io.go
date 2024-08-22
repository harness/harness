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

package storage

import (
	"context"
	"errors"
	"io"

	"github.com/harness/gitness/registry/app/driver"
)

const (
	maxBlobGetSize = 4 * 1024 * 1024
)

func getContent(ctx context.Context, driver driver.StorageDriver, p string) ([]byte, error) {
	r, err := driver.Reader(ctx, p, 0)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return readAllLimited(r, maxBlobGetSize)
}

func readAllLimited(r io.Reader, limit int64) ([]byte, error) {
	r = limitReader(r, limit)
	return io.ReadAll(r)
}

// limitReader returns a new reader limited to n bytes. Unlike io.LimitReader,
// this returns an error when the limit reached.
func limitReader(r io.Reader, n int64) io.Reader {
	return &limitedReader{r: r, n: n}
}

// limitedReader implements a reader that errors when the limit is reached.
//
// Partially cribbed from net/http.MaxBytesReader.
type limitedReader struct {
	r   io.Reader // underlying reader
	n   int64     // max bytes remaining
	err error     // sticky error
}

func (l *limitedReader) Read(p []byte) (n int, err error) {
	if l.err != nil {
		return 0, l.err
	}
	if len(p) == 0 {
		return 0, nil
	}
	// If they asked for a 32KB byte read but only 5 bytes are
	// remaining, no need to read 32KB. 6 bytes will answer the
	// question of the whether we hit the limit or go past it.
	if int64(len(p)) > l.n+1 {
		p = p[:l.n+1]
	}
	n, err = l.r.Read(p)

	if int64(n) <= l.n {
		l.n -= int64(n)
		l.err = err
		return n, err
	}

	n = int(l.n)
	l.n = 0

	l.err = errors.New("storage: read exceeds limit")
	return n, l.err
}
