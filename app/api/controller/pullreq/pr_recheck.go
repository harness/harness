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

package pullreq

import (
	"context"

	"github.com/harness/gitness/app/auth"
)

// Recheck re-checks all system PR checks (mergeability check, ...).
func (c *Controller) Recheck(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
) error {
	// TODO: Remove the API.
	_ = ctx
	_ = session
	_ = repoRef
	_ = prNum
	/*
		repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush)
		if err != nil {
			return fmt.Errorf("failed to acquire access to repo: %w", err)
		}

		err = c.pullreqService.UpdateMergeDataIfRequired(ctx, repo.ID, prNum)
		if err != nil {
			return fmt.Errorf("failed to refresh merge data: %w", err)
		}
	*/

	return nil
}
