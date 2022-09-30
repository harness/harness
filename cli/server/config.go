// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package server

import (
	"github.com/harness/gitness/types"

	"github.com/kelseyhightower/envconfig"
)

// load returns the system configuration from the
// host environment.
func load() (*types.Config, error) {
	config := new(types.Config)
	// read the configuration from the environment and
	// populate the configuration structure.
	err := envconfig.Process("", config)
	return config, err
}
