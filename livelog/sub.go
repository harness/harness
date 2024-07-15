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
	"sync"

	"github.com/rs/zerolog/log"
)

type subscriber struct {
	sync.Mutex

	handler chan *Line
	closec  chan struct{}
	closed  bool
}

func (s *subscriber) publish(line *Line) {
	defer func() {
		r := recover()
		if r != nil {
			log.Debug().Msgf("publishing to closed subscriber")
		}
	}()

	s.Lock()
	defer s.Unlock()

	select {
	case <-s.closec:
	case s.handler <- line:
	default:
		// lines are sent on a buffered channel. If there
		// is a slow consumer that is not processing events,
		// the buffered channel will fill and newer messages
		// are ignored.
	}
}

func (s *subscriber) close() {
	s.Lock()
	if !s.closed {
		close(s.closec)
		close(s.handler)
		s.closed = true
	}
	s.Unlock()
}
