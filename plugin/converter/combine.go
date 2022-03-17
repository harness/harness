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

package converter

import (
	"context"

	"github.com/drone/drone/core"
)

// Combine combines the conversion services, provision support
// for multiple conversion utilities.
func Combine(multi bool, services ...core.ConvertService) core.ConvertService {
	return &combined{multi: multi, sources: services}
}

type combined struct {
	sources []core.ConvertService

	// this feature flag can be removed once we solve for
	// https://github.com/harness/drone/pull/2994#issuecomment-795955312
	multi bool
}

func (c *combined) Convert(ctx context.Context, req *core.ConvertArgs) (*core.Config, error) {
	for _, source := range c.sources {
		config, err := source.Convert(ctx, req)
		if err != nil {
			return nil, err
		}
		if config == nil {
			continue
		}
		if config.Data == "" {
			continue
		}
		if c.multi {
			req.Config = config
		} else {
			return config, nil
		}
	}
	return req.Config, nil
}
