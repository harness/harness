// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package events

import (
	"github.com/harness/gitness/events"
)

func NewReaderFactory(eventsSystem *events.System) (*events.ReaderFactory[*Reader], error) {
	readerFactoryFunc := func(innerReader *events.GenericReader) (*Reader, error) {
		return &Reader{
			innerReader: innerReader,
		}, nil
	}

	return events.NewReaderFactory(eventsSystem, category, readerFactoryFunc)
}

// Reader is the event reader for this package.
type Reader struct {
	innerReader *events.GenericReader
}

func (r *Reader) Configure(opts ...events.ReaderOption) {
	r.innerReader.Configure(opts...)
}
