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

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ListChecks return an array of status check results for a commit in a repository.
func (c *Controller) ListChecks(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	commitSHA string,
	opts types.CheckListOptions,
) ([]types.Check, int, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	var checks []types.Check
	var count int

	err = c.tx.WithTx(ctx, func(ctx context.Context) (err error) {
		checks, err = c.checkStore.List(ctx, repo.ID, commitSHA, opts)
		if err != nil {
			return fmt.Errorf("failed to list status check results for repo=%s: %w", repo.Identifier, err)
		}

		if opts.Page == 1 && len(checks) < opts.Size {
			count = len(checks)
			return nil
		}

		count, err = c.checkStore.Count(ctx, repo.ID, commitSHA, opts)
		if err != nil {
			return fmt.Errorf("failed to count status check results for repo=%s: %w", repo.Identifier, err)
		}

		return nil
	})
	if err != nil {
		return nil, 0, err
	}

	return checks, count, nil
}
