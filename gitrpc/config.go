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

package gitrpc

import (
	"errors"
)

// Config represents the config for the gitrpc client.
type Config struct {
	Addr                string `envconfig:"GITRPC_CLIENT_ADDR" default:"127.0.0.1:3001"`
	LoadBalancingPolicy string `envconfig:"GITRPC_CLIENT_LOAD_BALANCING_POLICY" default:"pick_first"`
}

func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config is required")
	}
	if c.Addr == "" {
		return errors.New("config.Addr is required")
	}

	return nil
}
