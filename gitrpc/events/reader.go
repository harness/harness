// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package events

import (
	"time"

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
// It exposes typesafe event registration methods for all events by this package.
// NOTE: Event registration methods are in the event's dedicated file.
type Reader struct {
	innerReader *events.GenericReader
}

func (r *Reader) SetConcurrency(concurrency int) error {
	return r.innerReader.SetConcurrency(concurrency)
}

func (r *Reader) SetProcessingTimeout(timeout time.Duration) error {
	return r.innerReader.SetProcessingTimeout(timeout)
}
