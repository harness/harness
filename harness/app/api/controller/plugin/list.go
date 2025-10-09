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

package plugin

import (
	"context"
	"fmt"

	"github.com/harness/gitness/types"
)

// List lists all the global plugins.
// Since this just lists the schema of plugins, it does not require any
// specific authorization. Plugins are available globally so they are not
// associated with any space.
func (c *Controller) List(
	ctx context.Context,
	filter types.ListQueryFilter,
) ([]*types.Plugin, int64, error) {
	plugins, err := c.pluginStore.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list plugins: %w", err)
	}

	if len(plugins) < filter.Size {
		return plugins, int64(len(plugins)), nil
	}
	count, err := c.pluginStore.Count(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count plugins: %w", err)
	}

	return plugins, count, nil
}
