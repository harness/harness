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

package server

import (
	"strings"

	"github.com/harness/gitness/types"

	"github.com/go-redis/redis/v8"
)

// ProvideRedis provides a redis client based on the configuration.
// TODO: add support for TLS.
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
