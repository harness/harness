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
	"context"
	"io"
	"net/http"
)

type writeCounter struct {
	ctx       context.Context
	w         http.ResponseWriter
	spaceRef  string
	intf      Sender
	isStorage bool
}

func newWriter(
	ctx context.Context,
	w http.ResponseWriter,
	spaceRef string,
	intf Sender,
	isStorage bool,
) *writeCounter {
	return &writeCounter{
		ctx:       ctx,
		w:         w,
		spaceRef:  spaceRef,
		intf:      intf,
		isStorage: isStorage,
	}
}

func (c *writeCounter) Write(data []byte) (n int, err error) {
	n, err = c.w.Write(data)

	m := Metric{
		SpaceRef: c.spaceRef,
		Size: Size{
			Bandwidth: int64(n),
		},
	}
	if c.isStorage {
		m.Storage = int64(n)
	}

	sendErr := c.intf.Send(c.ctx, m)
	if sendErr != nil {
		return n, sendErr
	}

	return n, err
}

func (c *writeCounter) Header() http.Header {
	return c.w.Header()
}

func (c *writeCounter) WriteHeader(statusCode int) {
	c.w.WriteHeader(statusCode)
}

type readCounter struct {
	ctx       context.Context
	r         io.ReadCloser
	spaceRef  string
	intf      Sender
	isStorage bool
}

func newReader(
	ctx context.Context,
	r io.ReadCloser,
	spaceRef string,
	intf Sender,
	isStorage bool,
) *readCounter {
	return &readCounter{
		ctx:       ctx,
		r:         r,
		spaceRef:  spaceRef,
		intf:      intf,
		isStorage: isStorage,
	}
}

func (c *readCounter) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)

	m := Metric{
		SpaceRef: c.spaceRef,
		Size: Size{
			Bandwidth: int64(n),
		},
	}
	if c.isStorage {
		m.Storage = int64(n)
	}

	sendErr := c.intf.Send(c.ctx, m)
	if sendErr != nil {
		return n, sendErr
	}

	return n, err
}

func (c *readCounter) Close() error {
	return c.r.Close()
}
