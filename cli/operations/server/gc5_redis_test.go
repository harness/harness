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
	"os"
	"testing"

	"github.com/harness/gitness/types"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

func TestGc5ProvideRedisStandalone(t *testing.T) {
	cfg := &types.Config{}
	cfg.Redis.Endpoint = "127.0.0.1:6390"
	cfg.Redis.Password = "gc5-pass"
	cfg.Redis.MaxRetries = 4
	cfg.Redis.MinIdleConnections = 2
	cfg.Redis.MaxConnections = 15

	client, err := ProvideRedis(cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	plain, ok := client.(*redis.Client)
	require.True(t, ok)
	opts := plain.Options()
	require.Equal(t, "127.0.0.1:6390", opts.Addr)
	require.Equal(t, "gc5-pass", opts.Password)
	require.Equal(t, 4, opts.MaxRetries)
	require.Equal(t, 2, opts.MinIdleConns)
	require.Equal(t, 15, opts.PoolSize)
	require.Nil(t, opts.TLSConfig)
}

func TestGc5ProvideRedisSentinel(t *testing.T) {
	cfg := &types.Config{}
	cfg.Redis.SentinelMode = true
	cfg.Redis.SentinelMaster = "gc5-master"
	cfg.Redis.SentinelEndpoint = "127.0.0.1:26390,127.0.0.1:26391"
	cfg.Redis.Password = "gc5-pass"
	cfg.Redis.SSLEnabled = true

	client, err := ProvideRedis(cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	// the failover client is a *redis.Client too, so check that no standalone address was configured.
	failover, ok := client.(*redis.Client)
	require.True(t, ok)
	require.Equal(t, "FailoverClient", failover.Options().Addr)
	require.NotNil(t, failover.Options().TLSConfig)
}

func TestGc5ProvideRedisTLSErrors(t *testing.T) {
	cfg := &types.Config{}
	cfg.Redis.SSLEnabled = true
	cfg.Redis.CertPath = "/does/not/exist.pem"

	_, err := ProvideRedis(cfg)
	require.ErrorContains(t, err, "failed to read redis CA file")

	cfg.Redis.CertPath = t.TempDir() + "/ca.pem"
	require.NoError(t, gc5WriteFile(cfg.Redis.CertPath, "not a certificate"))

	_, err = ProvideRedis(cfg)
	require.ErrorContains(t, err, "failed to add certs")
}

func gc5WriteFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o600)
}
