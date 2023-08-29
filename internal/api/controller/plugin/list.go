// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.
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
