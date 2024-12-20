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

package space

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// GetUsageMetrics returns usage metrics for root space.
func (c *Controller) GetUsageMetrics(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	startDate int64,
	endDate int64,
) (*types.UsageMetric, error) {
	rootSpaceRef, _, err := paths.DisectRoot(spaceRef)
	if err != nil {
		return nil, fmt.Errorf("could not find root space: %w", err)
	}
	space, err := c.getSpaceCheckAuth(ctx, session, rootSpaceRef, enum.PermissionSpaceView)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to space: %w", err)
	}

	metric, err := c.usageMetricStore.GetMetrics(
		ctx,
		space.ID,
		startDate,
		endDate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve usage metrics: %w", err)
	}

	return metric, nil
}
