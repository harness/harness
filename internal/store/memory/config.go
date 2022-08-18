// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

// Package memory provides readonly memory data storage.
package memory

import (
	"context"

	"github.com/harness/scm/types"
)

// New returns a new system configuration store.
func New(config *types.Config) *SystemStore {
	return &SystemStore{config: config}
}

// SystemStore is a system store that loads system
// configuration parameters stored in the environment.
type SystemStore struct {
	config *types.Config
}

// Config returns the system configuration.
func (c *SystemStore) Config(ctx context.Context) *types.Config {
	return c.config
}
