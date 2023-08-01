// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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
