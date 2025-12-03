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

package stream

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

type RedisProducer struct {
	rdb redis.UniversalClient
	// namespace defines the namespace of the stream keys - any stream key will be prefixed with it.
	namespace string
	// maxStreamLength defines the maximum number of entries in each stream (ring buffer).
	maxStreamLength int64
	// approxMaxStreamLength specifies whether the maxStreamLength should be approximated.
	// NOTE: enabling approximation of stream length can lead to performance improvements.
	approxMaxStreamLength bool
}

func NewRedisProducer(rdb redis.UniversalClient, namespace string,
	maxStreamLength int64, approxMaxStreamLength bool) *RedisProducer {
	return &RedisProducer{
		rdb:                   rdb,
		namespace:             namespace,
		maxStreamLength:       maxStreamLength,
		approxMaxStreamLength: approxMaxStreamLength,
	}
}

// Send sends information to the Redis stream.
// Returns the message ID in case of success.
func (p *RedisProducer) Send(ctx context.Context, streamID string, payload map[string]interface{}) (string, error) {
	// ensure we transpose streamID using the key namespace
	transposedStreamID := transposeStreamID(p.namespace, streamID)

	// send message to stream - will create the stream if it doesn't exist yet
	// NOTE: response is the message ID (See https://redis.io/commands/xadd/)
	args := &redis.XAddArgs{
		Stream: transposedStreamID,
		Values: payload,
		MaxLen: p.maxStreamLength,
		Approx: p.approxMaxStreamLength,
		ID:     "*", // let redis create message ID
	}
	msgID, err := p.rdb.XAdd(ctx, args).Result()
	if err != nil {
		return "", fmt.Errorf("failed to write to stream '%s' (redis stream '%s'). Error: %w",
			streamID, transposedStreamID, err)
	}

	return msgID, nil
}
