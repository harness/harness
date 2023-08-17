// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package livelog

import (
	"context"
	"sync"
)

// this is the amount of items that are stored in memory
// in the buffer. This should result in approximately 10kb
// of memory allocated per-stream and per-subscriber, not
// including any logdata stored in these structures.
const bufferSize = 5000

type stream struct {
	sync.Mutex

	hist []*Line
	list map[*subscriber]struct{}
}

func newStream() *stream {
	return &stream{
		list: map[*subscriber]struct{}{},
	}
}

func (s *stream) write(line *Line) error {
	s.Lock()
	s.hist = append(s.hist, line)
	for l := range s.list {
		l.publish(line)
	}
	// the history should not be unbounded. The history
	// slice is capped and items are removed in a FIFO
	// ordering when capacity is reached.
	if size := len(s.hist); size >= bufferSize {
		s.hist = s.hist[size-bufferSize:]
	}
	s.Unlock()
	return nil
}

func (s *stream) subscribe(ctx context.Context) (<-chan *Line, <-chan error) {
	sub := &subscriber{
		handler: make(chan *Line, bufferSize),
		closec:  make(chan struct{}),
	}
	err := make(chan error)

	s.Lock()
	for _, line := range s.hist {
		sub.publish(line)
	}
	s.list[sub] = struct{}{}
	s.Unlock()

	go func() {
		defer close(err)
		select {
		case <-sub.closec:
		case <-ctx.Done():
			sub.close()
		}
	}()
	return sub.handler, err
}

func (s *stream) close() error {
	s.Lock()
	defer s.Unlock()
	for sub := range s.list {
		delete(s.list, sub)
		sub.close()
	}
	return nil
}
