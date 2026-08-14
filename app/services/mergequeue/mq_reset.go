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
	"time"

	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

// reset resets all merge queue entries (sets state of the entries to "merge pending").
func (s *Service) reset(ctx context.Context, repo *types.RepositoryCore, q *types.MergeQueue) {
	unlock, err := s.locker.LockBranch(
		ctx,
		q.RepoID,
		q.Branch,
		30*time.Second,
	)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).
			Int64("repo_id", q.RepoID).
			Str("branch", q.Branch).
			Msg("failed to lock repository for merge queue reset")

		return
	}
	defer unlock()

	entries, err := s.mergeQueueEntryStore.ListForMergeQueue(ctx, q.ID)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).
			Int64("repo_id", q.RepoID).
			Str("branch", q.Branch).
			Msg("failed to list all merge queue entries")

		return
	}

	s.resetEntries(ctx, q, entries)

	s.deleteReference(ctx, repo, q.Branch)
}
