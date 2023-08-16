// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package server

import (
	"errors"
	"time"
)

const (
	ModeInMemory = "inmemory"
	ModeRedis    = "redis"
	ModeNone     = "none"
)

// Config represents the configuration for the gitrpc server.
type Config struct {
	// Bind specifies the addr used to bind the grpc server.
	Bind string `envconfig:"GITRPC_SERVER_BIND" default:":3001"`
	// GitRoot specifies the directory containing git related data (e.g. repos, ...)
	GitRoot string `envconfig:"GITRPC_SERVER_GIT_ROOT"`
	// TmpDir (optional) specifies the directory for temporary data (e.g. repo clones, ...)
	TmpDir string `envconfig:"GITRPC_SERVER_TMP_DIR"`
	// GitHookPath points to the binary used as git server hook.
	GitHookPath string `envconfig:"GITRPC_SERVER_GIT_HOOK_PATH"`

	HTTP struct {
		Bind string `envconfig:"GITRPC_SERVER_HTTP_BIND" default:":4001"`
	}
	MaxConnAge      time.Duration `envconfig:"GITRPC_SERVER_MAX_CONN_AGE" default:"630720000s"`
	MaxConnAgeGrace time.Duration `envconfig:"GITRPC_SERVER_MAX_CONN_AGE_GRACE" default:"630720000s"`

	// LastCommitCache holds configuration options for the last commit cache.
	LastCommitCache struct {
		// Mode determines where the cache will be. Valid values are "inmemory" (default), "redis" or "none".
		Mode string `envconfig:"GITRPC_LAST_COMMIT_CACHE_MODE" default:"inmemory"`

		// DurationSeconds defines cache duration in seconds of last commit, default=12h.
		DurationSeconds int `envconfig:"GITRPC_LAST_COMMIT_CACHE_SECONDS" default:"43200"`
	}

	Redis struct {
		Endpoint           string `envconfig:"GITRPC_REDIS_ENDPOINT"             default:"localhost:6379"`
		MaxRetries         int    `envconfig:"GITRPC_REDIS_MAX_RETRIES"          default:"3"`
		MinIdleConnections int    `envconfig:"GITRPC_REDIS_MIN_IDLE_CONNECTIONS" default:"0"`
		Password           string `envconfig:"GITRPC_REDIS_PASSWORD"`
		SentinelMode       bool   `envconfig:"GITRPC_REDIS_USE_SENTINEL"         default:"false"`
		SentinelMaster     string `envconfig:"GITRPC_REDIS_SENTINEL_MASTER"`
		SentinelEndpoint   string `envconfig:"GITRPC_REDIS_SENTINEL_ENDPOINT"`
	}
}

func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config is required")
	}
	if c.Bind == "" {
		return errors.New("config.Bind is required")
	}
	if c.GitRoot == "" {
		return errors.New("config.GitRoot is required")
	}
	if c.GitHookPath == "" {
		return errors.New("config.GitHookPath is required")
	}
	if c.MaxConnAge == 0 {
		return errors.New("config.MaxConnAge is required")
	}
	if c.MaxConnAgeGrace == 0 {
		return errors.New("config.MaxConnAgeGrace is required")
	}
	if m := c.LastCommitCache.Mode; m != "" && m != ModeInMemory && m != ModeRedis && m != ModeNone {
		return errors.New("config.LastCommitCache.Mode has unsupported value")
	}

	return nil
}
