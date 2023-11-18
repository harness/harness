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

package adapter

import (
	"time"

	"github.com/harness/gitness/cache"
	"github.com/harness/gitness/git/types"
	apptypes "github.com/harness/gitness/types"

	"github.com/go-redis/redis/v8"
	"github.com/google/wire"
)

const (
	ModeInMemory = "inmemory"
	ModeRedis    = "redis"
	ModeNone     = "none"
)

var WireSet = wire.NewSet(
	ProvideGoGitRepoProvider,
	ProvideLastCommitCache,
)

func ProvideGoGitRepoProvider() *GoGitRepoProvider {
	const objectCacheSize = 16 << 20 // 16MiB
	return NewGoGitRepoProvider(objectCacheSize, 15*time.Minute)
}

func ProvideLastCommitCache(
	config *apptypes.Config,
	redisClient redis.UniversalClient,
	repoProvider *GoGitRepoProvider,
) cache.Cache[CommitEntryKey, *types.Commit] {
	cacheDuration := config.Git.LastCommitCache.Duration

	if config.Git.LastCommitCache.Mode == ModeNone || cacheDuration < time.Second {
		return NoLastCommitCache(repoProvider)
	}

	if config.Git.LastCommitCache.Mode == ModeRedis && redisClient != nil {
		return NewRedisLastCommitCache(redisClient, cacheDuration, repoProvider)
	}

	return NewInMemoryLastCommitCache(cacheDuration, repoProvider)
}
