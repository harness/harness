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

package queue

import (
	"context"
	"errors"

	"github.com/drone/drone/core"

	"github.com/go-redis/redis/v8"
)

type scheduler struct {
	*queue
	*canceller
}

type redisScheduler struct {
	*queue
	*redisCanceller
}

// New creates a new scheduler.
func New(store core.StageStore, rdb *redis.Client) core.Scheduler {
	if rdb != nil {
		return redisScheduler{
			queue:          newQueue(store),
			redisCanceller: newRedisCanceller(rdb),
		}
	}

	return scheduler{
		queue:     newQueue(store),
		canceller: newCanceller(),
	}
}

func (d scheduler) Stats(context.Context) (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (d redisScheduler) Stats(context.Context) (interface{}, error) {
	return nil, errors.New("not implemented")
}
