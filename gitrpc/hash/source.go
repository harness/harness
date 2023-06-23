// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package hash

import (
	"context"
	"fmt"
	"io"
)

// Source is an abstraction of a source of values that have to be hashed.
type Source interface {
	Next() ([]byte, error)
}

// SourceFunc is an alias for a function that returns the content of a source call by call.
type SourceFunc func() ([]byte, error)

func (f SourceFunc) Next() ([]byte, error) {
	return f()
}

// SourceFromSlice returns a source that iterates over the slice.
func SourceFromSlice(slice [][]byte) Source {
	return SourceFunc(func() ([]byte, error) {
		if len(slice) == 0 {
			return nil, io.EOF
		}

		// get next element and move slice forward
		next := slice[0]
		slice = slice[1:]

		return next, nil
	})
}

// SourceNext encapsulates the data that is needed to serve a call to Source.Next().
// It is being used by SourceFromChannel to expose a channel as Source.
type SourceNext struct {
	Data []byte
	Err  error
}

// SourceFromChannel creates a source that returns all elements read from nextChan.
// The .Data and .Err of a SourceNext object in the channel will be returned as is.
// If the channel is closed, the source indicates the end of the data.
func SourceFromChannel(ctx context.Context, nextChan <-chan SourceNext) Source {
	return SourceFunc(func() ([]byte, error) {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("source context failed with: %w", ctx.Err())
		case next, ok := <-nextChan:
			// channel closed, end of operation
			if !ok {
				return nil, io.EOF
			}

			return next.Data, next.Err
		}
	})
}
