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

	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"
)

func (s *Service) updateCodeCommentsOnBranchUpdate(ctx context.Context,
	event *events.Event[*pullreqevents.BranchUpdatedPayload],
) error {
	return s.updateCodeComments(ctx,
		event.Payload.TargetRepoID, event.Payload.PullReqID,
		event.Payload.NewSHA, event.Payload.NewMergeBaseSHA)
}

func (s *Service) updateCodeCommentsOnReopen(ctx context.Context,
	event *events.Event[*pullreqevents.ReopenedPayload],
) error {
	return s.updateCodeComments(ctx,
		event.Payload.TargetRepoID, event.Payload.PullReqID,
		event.Payload.SourceSHA, event.Payload.MergeBaseSHA)
}

func (s *Service) updateCodeComments(ctx context.Context,
	targetRepoID, pullreqID int64,
	newSourceSHA, newMergeBaseSHA string,
) error {
	repoGit, err := s.repoFinder.FindByID(ctx, targetRepoID)
	if err != nil {
		return fmt.Errorf("failed to get repo git info: %w", err)
	}

	var codeComments []*types.CodeComment

	codeComments, err = s.codeCommentView.ListNotAtMergeBaseSHA(ctx, pullreqID, newMergeBaseSHA)
	if err != nil {
		return fmt.Errorf("failed to get list of code comments for update after merge base update: %w", err)
	}

	s.codeCommentMigrator.MigrateOld(ctx, repoGit.GitUID, newMergeBaseSHA, codeComments)

	err = s.codeCommentView.UpdateAll(ctx, codeComments)
	if err != nil {
		return fmt.Errorf("failed to update code comments after merge base update: %w", err)
	}

	codeComments, err = s.codeCommentView.ListNotAtSourceSHA(ctx, pullreqID, newSourceSHA)
	if err != nil {
		return fmt.Errorf("failed to get list of code comments for update after source branch update: %w", err)
	}

	s.codeCommentMigrator.MigrateNew(ctx, repoGit.GitUID, newSourceSHA, codeComments)

	err = s.codeCommentView.UpdateAll(ctx, codeComments)
	if err != nil {
		return fmt.Errorf("failed to update code comments after source branch update: %w", err)
	}

	return nil
}
