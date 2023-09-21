// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/harness/gitness/gitrpc/server"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Debug bool `envconfig:"GITRPC_SERVER_DEBUG"`
	Trace bool `envconfig:"GITRPC_SERVER_TRACE"`

	// GracefulShutdownTime defines the max time we wait when shutting down a server.
	// 5min should be enough for most git clones to complete.
	GracefulShutdownTime time.Duration `envconfig:"GITRPC_SERVER_GRACEFUL_SHUTDOWN_TIME" default:"300s"`

	Profiler struct {
		Type        string `envconfig:"GITRPC_PROFILER_TYPE"`
		ServiceName string `envconfig:"GITRPC_PROFILER_SERVICE_NAME" default:"gitrpcserver"`
	}
}

func loadConfig() (Config, error) {
	config := Config{}
	err := envconfig.Process("", &config)
	if err != nil {
		return Config{}, fmt.Errorf("processing of config failed: %w", err)
	}
	return config, nil
}

func ProvideGitRPCServerConfig() (server.Config, error) {
	config := server.Config{}
	err := envconfig.Process("", &config)
	if err != nil {
		return server.Config{}, fmt.Errorf("processing of gitrpc server config failed: %w", err)
	}
	if config.GitRoot == "" {
		var homedir string
		homedir, err = os.UserHomeDir()
		if err != nil {
			return server.Config{}, err
		}

		config.GitRoot = filepath.Join(homedir, ".gitrpc")
	}

	return config, nil
}
