// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package server

import (
	"github.com/harness/gitness/types"

	"github.com/go-redis/redis/v8"
)

// ProvideRedis provides a redis client based on the configuration.
// TODO: add support for sentinal / cluster
// TODO: add support for TLS
func ProvideRedis(config *types.Config) (redis.UniversalClient, error) {
	options := &redis.Options{
		Addr:         config.Redis.Endpoint,
		MaxRetries:   config.Redis.MaxRetries,
		MinIdleConns: config.Redis.MinIdleConnections,
	}

	if config.Redis.Password != "" {
		options.Password = config.Redis.Password
	}

	return redis.NewClient(options), nil
}
