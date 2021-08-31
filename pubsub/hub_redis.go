// Copyright 2021 Drone IO, Inc.
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

// +build !oss

package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/drone/drone/core"
	"github.com/drone/drone/service/redisdb"
)

const (
	redisPubSubEvents   = "drone-events"
	redisPubSubCapacity = 100
)

func newHubRedis(r redisdb.RedisDB) core.Pubsub {
	h := &hubRedis{
		rdb:         r,
		subscribers: make(map[chan<- *core.Message]struct{}),
	}

	go r.Subscribe(context.Background(), redisPubSubEvents, redisPubSubCapacity, h)

	return h
}

type hubRedis struct {
	sync.Mutex
	rdb         redisdb.RedisDB
	subscribers map[chan<- *core.Message]struct{}
}

// Publish publishes a new message. All subscribers will get it.
func (h *hubRedis) Publish(ctx context.Context, e *core.Message) (err error) {
	client := h.rdb.Client()

	data, err := json.Marshal(e)
	if err != nil {
		return
	}

	_, err = client.Publish(ctx, redisPubSubEvents, data).Result()
	if err != nil {
		return
	}

	return
}

// Subscribe add a new subscriber. The subscriber gets event until its context is not finished.
func (h *hubRedis) Subscribe(ctx context.Context) (<-chan *core.Message, <-chan error) {
	chMessage := make(chan *core.Message, redisPubSubCapacity)
	chErr := make(chan error)

	h.Lock()
	h.subscribers[chMessage] = struct{}{}
	h.Unlock()

	go func() {
		<-ctx.Done()

		h.Lock()
		delete(h.subscribers, chMessage)
		h.Unlock()

		close(chMessage)
		close(chErr)
	}()

	return chMessage, chErr
}

// Subscribers returns number of subscribers.
func (h *hubRedis) Subscribers() (int, error) {
	h.Lock()
	n := len(h.subscribers)
	h.Unlock()

	return n, nil
}

// ProcessMessage relays the message to all subscribers listening to drone events.
// It is a part of redisdb.PubSubProcessor implementation and it's called internally by redisdb.Subscribe.
func (h *hubRedis) ProcessMessage(s string) {
	message := &core.Message{}
	err := json.Unmarshal([]byte(s), message)
	if err != nil {
		// Ignore invalid messages. This is a "should not happen" situation,
		// because messages are encoded as json in Publish().
		_, _ = fmt.Fprintf(os.Stderr, "pubsub/redis: failed to unmarshal a message. %s\n", err)
		return
	}

	h.Lock()
	for ss := range h.subscribers {
		select {
		case ss <- message:
		default: // messages are lost if a subscriber channel reaches its capacity
		}
	}
	h.Unlock()
}

// ProcessError is a part of redisdb.PubSubProcessor implementation.
func (h *hubRedis) ProcessError(error) {}
