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

package streamio

import "io"

const MaxBufferSize = 128 * 1024

type writer struct {
	bufferSize int
	sender     func([]byte) error
}

type Option func(w *writer)

func WithBufferSize(size int) Option {
	return func(w *writer) {
		w.bufferSize = size
	}
}

func NewWriter(sender func(p []byte) error, options ...Option) io.Writer {
	w := &writer{
		sender: sender,
	}

	for _, option := range options {
		option(w)
	}

	if w.bufferSize == 0 || w.bufferSize > MaxBufferSize {
		w.bufferSize = MaxBufferSize
	}

	return w
}

func (w *writer) Write(p []byte) (int, error) {
	var sent int

	for len(p) > 0 {
		chunkSize := len(p)
		if chunkSize > w.bufferSize {
			chunkSize = w.bufferSize
		}

		if err := w.sender(p[:chunkSize]); err != nil {
			return sent, err
		}

		sent += chunkSize
		p = p[chunkSize:]
	}

	return sent, nil
}

func NewReader(receiver func() ([]byte, error)) io.Reader {
	return &reader{receiver: receiver}
}

type reader struct {
	receiver func() ([]byte, error)
	data     []byte
	err      error
}

func (r *reader) Read(p []byte) (int, error) {
	if len(r.data) == 0 && r.err == nil {
		r.data, r.err = r.receiver()
	}

	n := copy(p, r.data)
	r.data = r.data[n:]

	if len(r.data) == 0 {
		return n, r.err
	}

	return n, nil
}
