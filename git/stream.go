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

package git

import "io"

// StreamReader is a helper utility to ease reading from streaming channel pair (the data and the error channel).
type StreamReader[T any] struct {
	chData <-chan T
	chErr  <-chan error
}

// NewStreamReader creates new StreamReader.
func NewStreamReader[T any](chData <-chan T, chErr <-chan error) *StreamReader[T] {
	return &StreamReader[T]{
		chData: chData,
		chErr:  chErr,
	}
}

// Next returns the next element or error.
// In case the end has been reached, an io.EOF is returned.
func (str *StreamReader[T]) Next() (T, error) {
	var null T

	select {
	case data, ok := <-str.chData:
		if !ok {
			return null, io.EOF
		}

		return data, nil
	case err, ok := <-str.chErr:
		if !ok {
			return null, io.EOF
		}

		return null, err
	}
}
