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
	"github.com/drone/drone/core"

	"github.com/go-redis/redis/v8"
)

// New creates a new publish subscriber. If Redis client passed as parameter is not nil it uses
// a Redis implementation, otherwise it uses an in-memory implementation.
func New(rdb *redis.Client) (core.Pubsub, error) {
	if rdb != nil {
		return newRedis(rdb)
	}

	return newHub(), nil
}
