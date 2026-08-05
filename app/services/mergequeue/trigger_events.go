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

	mergequeueevents "github.com/harness/gitness/app/events/mergequeue"
)

// TriggerBranchDeleted triggers merge queue branch deleted event. Intended to be called from the post receive githook.
func (s *Service) TriggerBranchDeleted(ctx context.Context, repoID int64, branch string) {
	s.mergeQueueEventReporter.BranchDeleted(ctx, &mergequeueevents.BranchDeletedPayload{
		Base: mergequeueevents.Base{
			RepoID: repoID,
			Branch: branch,
		},
	})
}

// TriggerBranchUpdated triggers merge queue branch updated event. Intended to be called from the post receive githook.
func (s *Service) TriggerBranchUpdated(ctx context.Context, repoID int64, branch string) {
	// The target branch was changed outside the merge queue. Publish it as a generic update event so that
	// handlerUpdated reprocesses the queue: reprocess re-reads the live target SHA and recreates the merge commit
	// for every entry whose base no longer matches, which self-heals the queue after the direct change.
	s.mergeQueueEventReporter.Updated(ctx, &mergequeueevents.UpdatedPayload{
		Base: mergequeueevents.Base{
			RepoID: repoID,
			Branch: branch,
		},
	})
}
