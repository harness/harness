// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package server

import (
	"strings"

	"github.com/harness/gitness/types"

	"github.com/go-redis/redis/v8"
)

// ProvideRedis provides a redis client based on the configuration.
// TODO: add support for TLS
func ProvideRedis(config *types.Config) (redis.UniversalClient, error) {
	if config.Redis.SentinelMode {
		addrs := strings.Split(config.Redis.SentinelEndpoint, ",")

		failoverOptions := &redis.FailoverOptions{
			MasterName:    config.Redis.SentinelMaster,
			SentinelAddrs: addrs,
			MaxRetries:    config.Redis.MaxRetries,
			MinIdleConns:  config.Redis.MinIdleConnections,
		}
		if config.Redis.Password != "" {
			failoverOptions.Password = config.Redis.Password
		}
		return redis.NewFailoverClient(failoverOptions), nil
	}

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
