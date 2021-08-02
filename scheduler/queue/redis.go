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

package queue

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	redisPubSubCancel       = "drone-cancel"
	redisCancelValuePrefix  = "drone-cancel-"
	redisCancelValueTimeout = 5 * time.Minute
	redisCancelValue        = "canceled"
)

func newRedisCanceller(rdb *redis.Client) *redisCanceller {
	return &redisCanceller{rdb: rdb}
}

type redisCanceller struct {
	rdb *redis.Client
}

func (c *redisCanceller) Cancel(ctx context.Context, id int64) (err error) {
	ids := strconv.FormatInt(id, 10)

	// publish a cancel event to all subscribers (runners) waiting to
	_, err = c.rdb.Publish(ctx, redisPubSubCancel, ids).Result()
	if err != nil {
		return
	}

	// put a limited duration value in case a runner isn't listening currently.
	_, err = c.rdb.Set(ctx, redisCancelValuePrefix+ids, redisCancelValue, redisCancelValueTimeout).Result()
	if err != nil {
		return
	}

	return nil
}

func (c *redisCanceller) Cancelled(ctx context.Context, id int64) (isCancelled bool, err error) {
	ids := strconv.FormatInt(id, 10)

	// first check if the build is already cancelled

	result, err := c.rdb.Get(ctx, redisCancelValuePrefix+ids).Result()
	if err != nil && err != redis.Nil {
		return
	}

	isCancelled = err != redis.Nil && result == redisCancelValue
	if isCancelled {
		return
	}

	// if it is not cancelled, subscribe and listen to cancel build events
	// until the context is cancelled or until the build is cancelled.

	chResult := make(chan interface{})

	go func() {
		pubsub := c.rdb.Subscribe(ctx, redisPubSubCancel)
		ch := pubsub.Channel()

		defer func() {
			_ = pubsub.Close()
			close(chResult)
		}()

		err := pubsub.Ping(ctx)
		if err != nil {
			chResult <- err
			return
		}

		for {
			select {
			case m, ok := <-ch:
				if !ok {
					chResult <- fmt.Errorf("canceller/redis: channel=%s closed", redisPubSubCancel)
					return
				}

				idMessage, err := strconv.ParseInt(m.Payload, 10, 64)
				if err != nil { // should not happen
					_, _ = fmt.Fprintf(os.Stderr, "canceller/redis: message is not an integer: %s\n", m.Payload)
					continue // ignore data errors
				}

				if id == idMessage {
					chResult <- true
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	value, ok := <-chResult

	if !ok {
		return
	}

	err, ok = value.(error)
	if ok {
		return
	}

	isCancelled, ok = value.(bool)
	if ok {
		return
	}

	return
}
