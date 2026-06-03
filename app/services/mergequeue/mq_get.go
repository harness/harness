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

package mergequeue

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
)

// FindOrCreateMergeQueue finds the merge queue for the given repo and branch,
// creating it if one does not yet exist. Do not call inside a transaction.
func (s *Service) FindOrCreateMergeQueue(
	ctx context.Context,
	repoID int64,
	branch string,
) (*types.MergeQueue, error) {
	q, err := s.mergeQueueStore.FindByRepoAndBranch(ctx, repoID, branch)
	if err != nil && !errors.Is(err, store.ErrResourceNotFound) {
		return nil, err
	}

	if q != nil {
		return q, nil
	}

	now := time.Now().UnixMilli()
	q = &types.MergeQueue{
		RepoID:  repoID,
		Branch:  branch,
		Created: now,
		Updated: now,
	}

	err = s.mergeQueueStore.Create(ctx, q)
	if errors.IsConflict(err) {
		// Handle a race where another request created it concurrently.
		q, err = s.mergeQueueStore.FindByRepoAndBranch(ctx, repoID, branch)
		if err != nil {
			return nil, fmt.Errorf("failed to find merge queue after conflict: %w", err)
		}

		return q, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create merge queue: %w", err)
	}

	return q, nil
}

func (s *Service) getLastCommitSHA(ctx context.Context, repo *types.RepositoryCore, branch string) (sha.SHA, error) {
	result, err := s.git.GetRef(ctx, git.GetRefParams{
		ReadParams: git.ReadParams{RepoUID: repo.GitUID},
		Name:       branch,
		Type:       gitenum.RefTypeBranch,
	})
	if err != nil {
		return sha.None, err
	}

	return result.SHA, nil
}
