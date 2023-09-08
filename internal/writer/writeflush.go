// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package writer

import "io"

type Flusher interface {
	Flush()
}

type writeWithFlusher struct {
	writer  io.Writer
	flusher Flusher
}

type WriterFlusher interface {
	io.Writer
	Flusher
}

func NewWriterFlusher(writer io.Writer, flusher Flusher) WriterFlusher {
	return &writeWithFlusher{
		writer:  writer,
		flusher: flusher,
	}
}

func (w *writeWithFlusher) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	return n, err
}

func (w *writeWithFlusher) Flush() {
	w.flusher.Flush()
}
