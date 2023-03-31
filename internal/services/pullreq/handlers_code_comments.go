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

	"github.com/rs/zerolog/log"
)

func (s *Service) updateCodeCommentsOnBranchUpdate(ctx context.Context,
	event *events.Event[*pullreqevents.BranchUpdatedPayload],
) error {
	oldSourceSHA := event.Payload.OldSHA // NOTE: we're ignoring the old value and instead try to update all
	newSourceSHA := event.Payload.NewSHA

	log.Ctx(ctx).Debug().
		Str("oldSHA", oldSourceSHA).
		Str("newSHA", newSourceSHA).
		Msgf("code comment update after source branch update")

	repoGit, err := s.repoGitInfoCache.Get(ctx, event.Payload.SourceRepoID)
	if err != nil {
		return fmt.Errorf("failed to get repo git info: %w", err)
	}

	codeComments, err := s.codeCommentView.ListNotAtSourceSHA(ctx, event.Payload.PullReqID, newSourceSHA)
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

func (s *Service) updateCodeCommentsOnMergeBaseUpdate(ctx context.Context,
	pr *types.PullReq,
	gitUID string,
	oldMergeBaseSHA, newMergeBaseSHA string,
) error {
	log.Ctx(ctx).Debug().
		Str("oldSHA", oldMergeBaseSHA).
		Str("newSHA", newMergeBaseSHA).
		Msgf("code comment update after merge base update")

	codeComments, err := s.codeCommentView.ListNotAtMergeBaseSHA(ctx, pr.ID, newMergeBaseSHA)
	if err != nil {
		return fmt.Errorf("failed to get list of code comments for update after merge base update: %w", err)
	}

	s.codeCommentMigrator.MigrateOld(ctx, gitUID, newMergeBaseSHA, codeComments)

	err = s.codeCommentView.UpdateAll(ctx, codeComments)
	if err != nil {
		return fmt.Errorf("failed to update code comments after merge base update: %w", err)
	}

	return nil
}
