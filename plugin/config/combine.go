// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
