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
