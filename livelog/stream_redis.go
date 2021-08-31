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

// +build !oss

package livelog

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/drone/drone/core"
	"github.com/drone/drone/service/redisdb"

	"github.com/go-redis/redis/v8"
)

func newStreamRedis(r redisdb.RedisDB) core.LogStream {
	return streamRedis{
		rdb: r,
	}
}

const (
	redisKeyExpiryTime = 5 * time.Hour          // How long each key exists in redis
	redisPollTime      = 100 * time.Millisecond // should not be too large to avoid redis clients getting occupied for long
	redisTailMaxTime   = 1 * time.Hour          // maximum duration a tail can last
	redisEntryKey      = "line"
	redisStreamPrefix  = "drone-log-"
)

type streamRedis struct {
	rdb redisdb.RedisDB
}

// Create creates a redis stream and sets an expiry on it.
func (r streamRedis) Create(ctx context.Context, id int64) error {
	// Delete if a stream already exists with the same key
	_ = r.Delete(ctx, id)

	client := r.rdb.Client()

	key := redisStreamPrefix + strconv.FormatInt(id, 10)

	addResp := client.XAdd(ctx, &redis.XAddArgs{
		Stream: key,
		ID:     "*", // auto-generate a unique incremental ID
		MaxLen: bufferSize,
		Approx: true,
		Values: map[string]interface{}{redisEntryKey: []byte{}},
	})
	if err := addResp.Err(); err != nil {
		return fmt.Errorf("livelog/redis: could not create stream with key %s", key)
	}

	res := client.Expire(ctx, key, redisKeyExpiryTime)
	if err := res.Err(); err != nil {
		return fmt.Errorf("livelog/redis: could not set expiry for key %s", key)
	}

	return nil
}

// Delete deletes a stream
func (r streamRedis) Delete(ctx context.Context, id int64) error {
	client := r.rdb.Client()

	key := redisStreamPrefix + strconv.FormatInt(id, 10)

	if err := r._exists(ctx, key); err != nil {
		return err
	}

	deleteResp := client.Del(ctx, key)
	if err := deleteResp.Err(); err != nil {
		return fmt.Errorf("livelog/redis: could not delete stream for step %d", id)
	}

	return nil
}

// Write writes information into the Redis stream
func (r streamRedis) Write(ctx context.Context, id int64, line *core.Line) error {
	client := r.rdb.Client()

	key := redisStreamPrefix + strconv.FormatInt(id, 10)

	if err := r._exists(ctx, key); err != nil {
		return err
	}

	lineJsonData, _ := json.Marshal(line)
	addResp := client.XAdd(ctx, &redis.XAddArgs{
		Stream: key,
		ID:     "*", // auto-generate a unique incremental ID
		MaxLen: bufferSize,
		Approx: true,
		Values: map[string]interface{}{redisEntryKey: lineJsonData},
	})
	if err := addResp.Err(); err != nil {
		return err
	}

	return nil
}

// Tail returns back all the lines in the stream.
func (r streamRedis) Tail(ctx context.Context, id int64) (<-chan *core.Line, <-chan error) {
	client := r.rdb.Client()

	key := redisStreamPrefix + strconv.FormatInt(id, 10)

	if err := r._exists(ctx, key); err != nil {
		return nil, nil
	}

	chLines := make(chan *core.Line, bufferSize)
	chErr := make(chan error, 1)

	go func() {
		defer close(chErr)
		defer close(chLines)
		timeout := time.After(redisTailMaxTime) // polling should not last for longer than tailMaxTime

		// Keep reading from the stream and writing to the channel
		lastID := "0"

		for {
			select {
			case <-ctx.Done():
				return
			case <-timeout:
				return
			default:
				readResp := client.XRead(ctx, &redis.XReadArgs{
					Streams: append([]string{key}, lastID),
					Block:   redisPollTime, // periodically check for ctx.Done
				})
				if readResp.Err() != nil && readResp.Err() != redis.Nil { // readResp.Err() is sometimes set to "redis: nil" instead of nil
					chErr <- readResp.Err()
					return
				}

				for _, msg := range readResp.Val() {
					messages := msg.Messages
					if len(messages) > 0 {
						lastID = messages[len(messages)-1].ID
					} else { // should not happen
						return
					}

					for _, message := range messages {
						values := message.Values
						if val, ok := values[redisEntryKey]; ok {
							var line *core.Line
							if err := json.Unmarshal([]byte(val.(string)), &line); err != nil {
								continue // ignore errors in the stream
							}
							chLines <- line
						}
					}
				}
			}
		}
	}()

	return chLines, chErr
}

// Info returns info about log streams present in redis
func (r streamRedis) Info(ctx context.Context) (info *core.LogStreamInfo) {
	client := r.rdb.Client()

	info = &core.LogStreamInfo{
		Streams: make(map[int64]int),
	}

	keysResp := client.Keys(ctx, redisStreamPrefix+"*")
	if err := keysResp.Err(); err != nil {
		return
	}

	for _, key := range keysResp.Val() {
		ids := key[len(redisStreamPrefix):]
		id, err := strconv.ParseInt(ids, 10, 64)
		if err != nil {
			continue
		}

		lenResp := client.XLen(ctx, key)
		if err := lenResp.Err(); err != nil {
			continue
		}

		size := int(lenResp.Val())

		info.Streams[id] = size
	}

	return
}

func (r streamRedis) _exists(ctx context.Context, key string) error {
	client := r.rdb.Client()

	exists := client.Exists(ctx, key)
	if exists.Err() != nil || exists.Val() == 0 {
		return fmt.Errorf("livelog/redis: log stream %s not found", key)
	}

	return nil
}
