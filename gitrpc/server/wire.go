// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package server

import (
	"time"

	"github.com/harness/gitness/cache"
	"github.com/harness/gitness/gitrpc/internal/gitea"
	"github.com/harness/gitness/gitrpc/internal/service"
	"github.com/harness/gitness/gitrpc/internal/types"

	"github.com/go-redis/redis/v8"
	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideServer,
	ProvideHTTPServer,
	ProvideGITAdapter,
	ProvideGoGitRepoCache,
	ProvideLastCommitCache,
)

func ProvideGoGitRepoCache() cache.Cache[string, *gitea.RepoEntryValue] {
	return gitea.NewRepoCache()
}

func ProvideLastCommitCache(
	config Config,
	redisClient redis.UniversalClient,
	repoCache cache.Cache[string, *gitea.RepoEntryValue],
) cache.Cache[gitea.CommitEntryKey, *types.Commit] {
	cacheDuration := time.Duration(config.LastCommitCache.DurationSeconds) * time.Second

	if config.LastCommitCache.Mode == ModeNone || cacheDuration < time.Second {
		return gitea.NoLastCommitCache(repoCache)
	}

	if config.LastCommitCache.Mode == ModeRedis && redisClient != nil {
		return gitea.NewRedisLastCommitCache(redisClient, cacheDuration, repoCache)
	}

	return gitea.NewInMemoryLastCommitCache(cacheDuration, repoCache)
}

func ProvideGITAdapter(
	repoCache cache.Cache[string, *gitea.RepoEntryValue],
	lastCommitCache cache.Cache[gitea.CommitEntryKey, *types.Commit],
) (service.GitAdapter, error) {
	return gitea.New(repoCache, lastCommitCache)
}

func ProvideServer(config Config, adapter service.GitAdapter) (*GRPCServer, error) {
	return NewServer(config, adapter)
}

func ProvideHTTPServer(config Config) (*HTTPServer, error) {
	return NewHTTPServer(config)
}
