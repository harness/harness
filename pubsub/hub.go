// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pubsub

import (
	"context"
	"sync"

	"github.com/drone/drone/core"
)

type hub struct {
	sync.Mutex

	subs map[*subscriber]struct{}
}

// newHub creates a new publish subscriber.
func newHub() core.Pubsub {
	return &hub{
		subs: map[*subscriber]struct{}{},
	}
}

func (h *hub) Publish(ctx context.Context, e *core.Message) error {
	h.Lock()
	for s := range h.subs {
		s.publish(e)
	}
	h.Unlock()
	return nil
}

func (h *hub) Subscribe(ctx context.Context) (<-chan *core.Message, <-chan error) {
	h.Lock()
	s := &subscriber{
		handler: make(chan *core.Message, 100),
		quit:    make(chan struct{}),
	}
	h.subs[s] = struct{}{}
	h.Unlock()
	errc := make(chan error)
	go func() {
		defer close(errc)
		select {
		case <-ctx.Done():
			h.Lock()
			delete(h.subs, s)
			h.Unlock()
			s.close()
		}
	}()
	return s.handler, errc
}

func (h *hub) Subscribers() (int, error) {
	h.Lock()
	c := len(h.subs)
	h.Unlock()
	return c, nil
}
