// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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
