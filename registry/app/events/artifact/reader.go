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

package artifact

import (
	"github.com/harness/gitness/events"
)

func NewReaderFactory(eventsSystem *events.System) (*events.ReaderFactory[*Reader], error) {
	readerFactoryFunc := func(innerReader *events.GenericReader) (*Reader, error) {
		return &Reader{
			innerReader: innerReader,
		}, nil
	}

	return events.NewReaderFactory(eventsSystem, ArtifactsCategory, readerFactoryFunc)
}

// Reader is the event reader for this package.
// It exposes typesafe event registration methods for all events by this package.
// NOTE: Event registration methods are in the event's dedicated file.
type Reader struct {
	innerReader *events.GenericReader
}

func (r *Reader) Configure(opts ...events.ReaderOption) {
	r.innerReader.Configure(opts...)
}
