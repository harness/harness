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
	"github.com/drone/drone/core"
	"github.com/drone/drone/service/redisdb"
)

// New creates a new scheduler.
func New(store core.StageStore, r redisdb.RedisDB) core.Scheduler {
	if r == nil {
		return scheduler{
			queue:     newQueue(store),
			canceller: newCanceller(),
		}
	}

	return schedulerRedis{
		queue:          newQueue(store),
		cancellerRedis: newCancellerRedis(r),
	}
}
