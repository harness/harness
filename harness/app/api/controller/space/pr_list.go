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
)

// ListPullReqs returns a list of pull requests from the provided space.
func (c *Controller) ListPullReqs(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	includeSubspaces bool,
	filter *types.PullReqFilter,
) ([]types.PullReqRepo, error) {
	space, err := c.spaceFinder.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, fmt.Errorf("space not found: %w", err)
	}

	// We deliberately don't check for space permission because the pull request service
	// will check for repo-view permission for every returned pull request.

	pullReqs, err := c.prListService.ListForSpace(ctx, session, space, includeSubspaces, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pull requests from space: %w", err)
	}

	return pullReqs, nil
}
