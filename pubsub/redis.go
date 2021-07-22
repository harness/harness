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

package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/drone/drone/core"

	"github.com/go-redis/redis/v8"
)

const channelPubSub = "drone-events"

type hubRedis struct {
	rdb *redis.Client
}

func (h *hubRedis) Publish(ctx context.Context, e *core.Message) (err error) {
	data, err := json.Marshal(e)
	if err != nil {
		return
	}

	_, err = h.rdb.Publish(ctx, channelPubSub, data).Result()
	if err != nil {
		return
	}

	return
}

func (h *hubRedis) Subscribe(ctx context.Context) (<-chan *core.Message, <-chan error) {
	messageCh := make(chan *core.Message, 100)
	errCh := make(chan error)

	go func() {
		pubsub := h.rdb.Subscribe(ctx, channelPubSub)
		ch := pubsub.Channel(redis.WithChannelSize(100))

		defer func() {
			_ = pubsub.Close()
			close(messageCh)
			close(errCh)
		}()

		err := pubsub.Ping(ctx)
		if err != nil {
			errCh <- err
			return
		}

		for {
			select {
			case m, ok := <-ch:
				if ok {
					message := &core.Message{}
					err = json.Unmarshal([]byte(m.Payload), message)
					if err != nil {
						fmt.Printf("error@pubsub: failed to unmarshal a message. %s\n", err)
						continue
					}
					messageCh <- message
				} else {
					errCh <- fmt.Errorf("redis pubsub channel=%s closed", channelPubSub)
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return messageCh, errCh
}

func (h *hubRedis) Subscribers() int {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()

	v, err := h.rdb.Do(ctx, "pubsub", "numsub", channelPubSub).Result()
	if err != nil {
		fmt.Printf("error@pubsub: failed to get number of subscribers. %s\n", err)
		return -1
	}

	values, ok := v.([]interface{}) // the result should be: [<channel name, string>, <subscriber count, int64>]
	if !ok || len(values) != 2 {
		return 0
	}

	if subscriberCount, ok := values[1].(int64); ok {
		return int(subscriberCount)
	}

	return 0
}

func newRedis(rdb *redis.Client) (ps core.Pubsub, err error) {
	_, err = rdb.Ping(context.Background()).Result()
	if err != nil {
		err = fmt.Errorf("redis not accessibe: %w", err)
		return
	}

	ps = &hubRedis{
		rdb: rdb,
	}

	return
}
