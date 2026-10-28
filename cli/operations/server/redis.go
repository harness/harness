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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/harness/gitness/types"

	"github.com/go-redis/redis/v8"
)

// ProvideRedis provides a redis client based on the configuration.
func ProvideRedis(config *types.Config) (redis.UniversalClient, error) {
	var tlsConfig *tls.Config
	if config.Redis.SSLEnabled {
		var err error
		tlsConfig, err = redisTLSConfig(config)
		if err != nil {
			return nil, err
		}
	}

	if config.Redis.SentinelMode {
		addrs := strings.Split(config.Redis.SentinelEndpoint, ",")

		failoverOptions := &redis.FailoverOptions{
			MasterName:      config.Redis.SentinelMaster,
			SentinelAddrs:   addrs,
			MaxRetries:      config.Redis.MaxRetries,
			MinRetryBackoff: 100 * time.Millisecond,
			MaxRetryBackoff: 2 * time.Second,
			MinIdleConns:    config.Redis.MinIdleConnections,
			PoolSize:        config.Redis.MaxConnections,
			TLSConfig:       tlsConfig,
		}
		if config.Redis.Password != "" {
			failoverOptions.Password = config.Redis.Password
		}
		return redis.NewFailoverClient(failoverOptions), nil
	}

	options := &redis.Options{
		Addr:            config.Redis.Endpoint,
		MaxRetries:      config.Redis.MaxRetries,
		MinRetryBackoff: 100 * time.Millisecond,
		MaxRetryBackoff: 2 * time.Second,
		MinIdleConns:    config.Redis.MinIdleConnections,
		PoolSize:        config.Redis.MaxConnections,
		TLSConfig:       tlsConfig,
	}

	if config.Redis.Password != "" {
		options.Password = config.Redis.Password
	}

	return redis.NewClient(options), nil
}

// redisTLSConfig builds a TLS config. Caller invokes it only when SSL is enabled.
// An empty cert path uses the system pool.
func redisTLSConfig(config *types.Config) (*tls.Config, error) {
	if config.Redis.CertPath == "" {
		return &tls.Config{MinVersion: tls.VersionTLS12}, nil
	}

	certPool := x509.NewCertPool()
	pemCert, err := os.ReadFile(path.Clean(config.Redis.CertPath))
	if err != nil {
		return nil, fmt.Errorf("failed to read redis CA file: %w", err)
	}
	if !certPool.AppendCertsFromPEM(pemCert) {
		return nil, fmt.Errorf("failed to add certs from pem file %s to cert pool", config.Redis.CertPath)
	}
	return &tls.Config{
		RootCAs:    certPool,
		MinVersion: tls.VersionTLS12,
	}, nil
}
