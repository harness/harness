// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package config

import (
	"context"
	"errors"

	"github.com/drone/drone/core"
)

// error returned when no configured found.
var errNotFound = errors.New("configuration: not found")

// Combine combines the config services, allowing the system
// to source pipeline configuration from multiple sources.
func Combine(services ...core.ConfigService) core.ConfigService {
	return &combined{services}
}

type combined struct {
	sources []core.ConfigService
}

func (c *combined) Find(ctx context.Context, req *core.ConfigArgs) (*core.Config, error) {
	for _, source := range c.sources {
		config, err := source.Find(ctx, req)
		if err != nil {
			return nil, err
		}
		if config == nil {
			continue
		}
		if config.Data == "" {
			continue
		}
		return config, nil
	}
	return nil, errNotFound
}
