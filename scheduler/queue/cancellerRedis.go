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

package queue

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/drone/drone/service/redisdb"

	"github.com/go-redis/redis/v8"
)

const (
	redisPubSubCancel       = "drone-cancel"
	redisCancelValuePrefix  = "drone-cancel-"
	redisCancelValueTimeout = 5 * time.Minute
	redisCancelValue        = "canceled"
)

func newCancellerRedis(r redisdb.RedisDB) *cancellerRedis {
	h := &cancellerRedis{
		rdb:         r,
		subscribers: make(map[*cancelSubscriber]struct{}),
	}

	go r.Subscribe(context.Background(), redisPubSubCancel, 1, h)

	return h
}

type cancellerRedis struct {
	rdb         redisdb.RedisDB
	subscribers map[*cancelSubscriber]struct{}
	sync.Mutex
}

type cancelSubscriber struct {
	id int64
	ch chan<- error
}

// Cancel informs all subscribers that a build with the provided id is cancelled.
func (c *cancellerRedis) Cancel(ctx context.Context, id int64) (err error) {
	client := c.rdb.Client()

	ids := strconv.FormatInt(id, 10)

	// publish a cancel event to all subscribers (runners) waiting to
	_, err = client.Publish(ctx, redisPubSubCancel, ids).Result()
	if err != nil {
		return
	}

	// put a limited duration value in case a runner isn't listening currently.
	_, err = client.Set(ctx, redisCancelValuePrefix+ids, redisCancelValue, redisCancelValueTimeout).Result()
	if err != nil {
		return
	}

	return
}

// Cancelled waits until it gets info that a build with the provided id is cancelled.
// The waiting is aborted when the provided context is done.
func (c *cancellerRedis) Cancelled(ctx context.Context, id int64) (isCancelled bool, err error) {
	client := c.rdb.Client()

	ids := strconv.FormatInt(id, 10)

	// first check if the build is already cancelled

	result, err := client.Get(ctx, redisCancelValuePrefix+ids).Result()
	if err != nil && err != redis.Nil {
		return
	}

	isCancelled = err != redis.Nil && result == redisCancelValue
	if isCancelled {
		return
	}

	// if it is not cancelled, subscribe and listen to cancel build events
	// until the context is cancelled or until the build is cancelled.

	ch := make(chan error)
	sub := &cancelSubscriber{id: id, ch: ch}

	c.Lock()
	c.subscribers[sub] = struct{}{}
	c.Unlock()

	select {
	case err = <-ch:
		// If the build is cancelled or an error happened,
		// than the subscriber is removed from the set by other go routine
		isCancelled = err != nil
	case <-ctx.Done():
		// If the context is cancelled then the subscriber must be be removed here.
		c.Lock()
		delete(c.subscribers, sub)
		c.Unlock()
	}

	return
}

// ProcessMessage informs all subscribers listening to cancellation that the build with this id is cancelled.
// It is a part of redisdb.PubSubProcessor implementation and it's called internally by Subscribe.
func (c *cancellerRedis) ProcessMessage(s string) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		// Ignore invalid messages. This is a "should not happen" situation,
		// because all messages are integers as strings in method Cancel().
		_, _ = fmt.Fprintf(os.Stderr, "canceller/redis: message is not an integer: %s\n", s)
		return
	}

	c.Lock()
	for ss := range c.subscribers {
		if ss.id == id {
			ss.ch <- nil
			close(ss.ch)
			delete(c.subscribers, ss)
		}
	}
	c.Unlock()
}

// ProcessError informs all subscribers that an error happened and clears the set of subscribers.
// The set of subscribers is cleared because each subscriber receives only one message,
// so an error could cause that the message is missed - it's safer to return an error.
// It is a part of redisdb.PubSubProcessor implementation and it's called internally by Subscribe.
func (c *cancellerRedis) ProcessError(err error) {
	c.Lock()
	for ss := range c.subscribers {
		ss.ch <- err
		close(ss.ch)
		delete(c.subscribers, ss)
	}
	c.Unlock()
}
