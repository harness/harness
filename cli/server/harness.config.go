// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

// go:build harness

package server

import (
	"github.com/harness/gitness/harness/types"

	"github.com/kelseyhightower/envconfig"
)

func ProvideHarnessConfig() (*types.Config, error) {
	config := new(types.Config)
	err := envconfig.Process("", config)
	return config, err
}
