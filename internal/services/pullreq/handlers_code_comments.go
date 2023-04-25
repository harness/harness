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
	repoGit, err := s.repoGitInfoCache.Get(ctx, event.Payload.TargetRepoID)
	if err != nil {
		return fmt.Errorf("failed to get repo git info: %w", err)
	}

	var codeComments []*types.CodeComment

	newMergeBaseSHA := event.Payload.NewMergeBaseSHA

	codeComments, err = s.codeCommentView.ListNotAtMergeBaseSHA(ctx, event.Payload.PullReqID, newMergeBaseSHA)
	if err != nil {
		return fmt.Errorf("failed to get list of code comments for update after merge base update: %w", err)
	}

	s.codeCommentMigrator.MigrateOld(ctx, repoGit.GitUID, newMergeBaseSHA, codeComments)

	err = s.codeCommentView.UpdateAll(ctx, codeComments)
	if err != nil {
		return fmt.Errorf("failed to update code comments after merge base update: %w", err)
	}

	newSourceSHA := event.Payload.NewSHA

	codeComments, err = s.codeCommentView.ListNotAtSourceSHA(ctx, event.Payload.PullReqID, newSourceSHA)
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
