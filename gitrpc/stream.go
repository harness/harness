// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

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
