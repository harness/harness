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

package usage

import (
	"io"
	"net/http"
)

type writeCounter struct {
	w http.ResponseWriter
	n int64
}

func newWriter(w http.ResponseWriter) *writeCounter {
	return &writeCounter{
		w: w,
	}
}

func (c *writeCounter) Write(data []byte) (n int, err error) {
	n, err = c.w.Write(data)
	c.n += int64(n)
	return n, err
}

func (c *writeCounter) Header() http.Header {
	return c.w.Header()
}

func (c *writeCounter) WriteHeader(statusCode int) {
	c.w.WriteHeader(statusCode)
}

type readCounter struct {
	n int64
	r io.ReadCloser
}

func newReader(r io.ReadCloser) *readCounter {
	return &readCounter{
		r: r,
	}
}

func (c *readCounter) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}

func (c *readCounter) Close() error {
	return c.r.Close()
}
