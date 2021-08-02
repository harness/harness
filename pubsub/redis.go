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
	"os"
	"time"

	"github.com/drone/drone/core"

	"github.com/go-redis/redis/v8"
)

func newRedis(rdb *redis.Client) core.Pubsub {
	return &hubRedis{rdb: rdb}
}

const redisPubSubEvents = "drone-events"

type hubRedis struct {
	rdb *redis.Client
}

func (h *hubRedis) Publish(ctx context.Context, e *core.Message) (err error) {
	data, err := json.Marshal(e)
	if err != nil {
		return
	}

	_, err = h.rdb.Publish(ctx, redisPubSubEvents, data).Result()
	if err != nil {
		return
	}

	return
}

func (h *hubRedis) Subscribe(ctx context.Context) (<-chan *core.Message, <-chan error) {
	chMessage := make(chan *core.Message, 100)
	chErr := make(chan error)

	go func() {
		pubsub := h.rdb.Subscribe(ctx, redisPubSubEvents)
		ch := pubsub.Channel(redis.WithChannelSize(100))

		defer func() {
			_ = pubsub.Close()
			close(chMessage)
			close(chErr)
		}()

		err := pubsub.Ping(ctx)
		if err != nil {
			chErr <- err
			return
		}

		for {
			select {
			case m, ok := <-ch:
				if !ok {
					chErr <- fmt.Errorf("pubsub/redis: channel=%s closed", redisPubSubEvents)
					return
				}

				message := &core.Message{}
				err = json.Unmarshal([]byte(m.Payload), message)
				if err != nil {
					// This is a "should not happen" situation,
					// because messages are encoded as json above in Publish().
					_, _ = fmt.Fprintf(os.Stderr, "pubsub/redis: failed to unmarshal a message. %s\n", err)
					continue
				}

				chMessage <- message

			case <-ctx.Done():
				return
			}
		}
	}()

	return chMessage, chErr
}

func (h *hubRedis) Subscribers() (int, error) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()

	v, err := h.rdb.Do(ctx, "pubsub", "numsub", redisPubSubEvents).Result()
	if err != nil {
		err = fmt.Errorf("pubsub/redis: failed to get number of subscribers. %w", err)
		return 0, err
	}

	values, ok := v.([]interface{}) // the result should be: [<channel_name:string>, <subscriber_count:int64>]
	if !ok || len(values) != 2 {
		err = fmt.Errorf("pubsub/redis: failed to extarct number of subscribers from: %v", values)
		return 0, err
	}

	switch n := values[1].(type) {
	case int:
		return n, nil
	case uint:
		return int(n), nil
	case int32:
		return int(n), nil
	case uint32:
		return int(n), nil
	case int64:
		return int(n), nil
	case uint64:
		return int(n), nil
	default:
		err = fmt.Errorf("pubsub/redis: unsupported type for number of subscribers: %T", values[1])
		return 0, err
	}
}
