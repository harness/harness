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

package check

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ListRecentChecksSpace return an array of status check UIDs that have been run recently.
func (c *Controller) ListRecentChecksSpace(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	recursive bool,
	opts types.CheckRecentOptions,
) ([]string, error) {
	space, err := c.getSpaceCheckAccess(ctx, session, spaceRef, enum.PermissionSpaceEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to space: %w", err)
	}

	if opts.Since == 0 {
		opts.Since = time.Now().Add(-30 * 24 * time.Hour).UnixMilli()
	}

	var spaceIDs []int64
	if recursive {
		spaceIDs, err = c.spaceStore.GetDescendantsIDs(ctx, space.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get space descendants ids: %w", err)
		}
	} else {
		spaceIDs = append(spaceIDs, space.ID)
	}

	checkIdentifiers, err := c.checkStore.ListRecentSpace(ctx, spaceIDs, opts)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to list status check results for space=%s: %w",
			space.Identifier, err,
		)
	}

	return checkIdentifiers, nil
}
