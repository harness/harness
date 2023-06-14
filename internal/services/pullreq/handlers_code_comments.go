// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"
	"fmt"

	"github.com/harness/gitness/events"
	pullreqevents "github.com/harness/gitness/internal/events/pullreq"
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
	repoGit, err := s.repoGitInfoCache.Get(ctx, targetRepoID)
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
