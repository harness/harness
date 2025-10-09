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

package livelog

import (
	"context"
	"errors"
	"sync"
)

// ErrStreamNotFound is returned when a stream is not registered with
// the streamer.
var ErrStreamNotFound = errors.New("stream: not found")

type streamer struct {
	sync.Mutex

	streams map[int64]*stream
}

// New returns a new in-memory log streamer.
func NewMemory() LogStream {
	return &streamer{
		streams: make(map[int64]*stream),
	}
}

func (s *streamer) Create(_ context.Context, id int64) error {
	s.Lock()
	s.streams[id] = newStream()
	s.Unlock()
	return nil
}

func (s *streamer) Delete(_ context.Context, id int64) error {
	s.Lock()
	stream, ok := s.streams[id]
	if ok {
		delete(s.streams, id)
	}
	s.Unlock()
	if !ok {
		return ErrStreamNotFound
	}
	return stream.close()
}

func (s *streamer) Write(_ context.Context, id int64, line *Line) error {
	s.Lock()
	stream, ok := s.streams[id]
	s.Unlock()
	if !ok {
		return ErrStreamNotFound
	}
	return stream.write(line)
}

func (s *streamer) Tail(ctx context.Context, id int64) (<-chan *Line, <-chan error) {
	s.Lock()
	stream, ok := s.streams[id]
	s.Unlock()
	if !ok {
		return nil, nil
	}
	return stream.subscribe(ctx)
}

func (s *streamer) Info(_ context.Context) *LogStreamInfo {
	s.Lock()
	defer s.Unlock()
	info := &LogStreamInfo{
		Streams: map[int64]int{},
	}
	for id, stream := range s.streams {
		stream.Lock()
		info.Streams[id] = len(stream.list)
		stream.Unlock()
	}
	return info
}
