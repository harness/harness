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
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// CountPullReqs returns the number of pull requests from the provided space that matches the criteria.
func (c *Controller) CountPullReqs(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	includeSubspaces bool,
	filter *types.PullReqFilter,
) (int64, error) {
	space, err := c.getSpaceCheckAuth(ctx, session, spaceRef, enum.PermissionSpaceView)
	if err != nil {
		return 0, fmt.Errorf("failed to acquire access to space: %w", err)
	}

	count, err := c.prListService.CountForSpace(ctx, space, includeSubspaces, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch pull requests from space: %w", err)
	}

	return count, nil
}
