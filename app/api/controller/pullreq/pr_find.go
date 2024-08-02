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
	"fmt"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// Find returns a pull request from the provided repository.
func (c *Controller) Find(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pullreqNum int64,
) (*types.PullReq, error) {
	if pullreqNum <= 0 {
		return nil, usererror.BadRequest("A valid pull request number must be provided.")
	}

	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to the repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, pullreqNum)
	if err != nil {
		return nil, err
	}

	err = c.labelSvc.Backfill(ctx, pr)
	if err != nil {
		return nil, fmt.Errorf("failed to backfill labels assigned to pull request: %w", err)
	}

	if err := c.backfillStats(ctx, repo, pr); err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("failed to backfill PR stats")
	}

	return pr, nil
}

func (c *Controller) backfillStats(ctx context.Context, repo *types.Repository, pr *types.PullReq) error {
	s := pr.Stats.DiffStats
	if s.Commits != nil && s.FilesChanged != nil && s.Additions != nil && s.Deletions != nil {
		return nil
	}

	output, err := c.git.DiffStats(ctx, &git.DiffParams{
		ReadParams: git.CreateReadParams(repo),
		BaseRef:    pr.MergeBaseSHA,
		HeadRef:    pr.SourceSHA,
	})
	if err != nil {
		return fmt.Errorf("failed get diff stats: %w", err)
	}

	pr.Stats.DiffStats = types.NewDiffStats(output.Commits, output.FilesChanged, output.Additions, output.Deletions)

	return nil
}
