// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/events"
	"github.com/harness/gitness/gitrpc"
	gitrpcenum "github.com/harness/gitness/gitrpc/enum"
	gitevents "github.com/harness/gitness/internal/events/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

func (s *Service) handleEventBranchUpdated(ctx context.Context,
	event *events.Event[*gitevents.BranchUpdatedPayload],
) error {
	s.forEveryOpenPR(ctx, event.Payload.RepoID, event.Payload.Ref, func(pr *types.PullReq) error {
		targetRepo, err := s.repoStore.Find(ctx, pr.TargetRepoID)
		if err != nil {
			return fmt.Errorf("failed to get target repository after branch update: %w", err)
		}

		// TODO: This doesn't work for forked repos (only works when sourceRepo==targetRepo)
		err = s.gitRPCClient.UpdateRef(ctx, gitrpc.UpdateRefParams{
			WriteParams: gitrpc.WriteParams{RepoUID: targetRepo.GitUID},
			Name:        strconv.Itoa(int(pr.Number)),
			Type:        gitrpcenum.RefTypePullReqHead,
			NewValue:    event.Payload.NewSHA,
			OldValue:    event.Payload.OldSHA,
		})
		if err != nil {
			return fmt.Errorf("failed to update PR head ref after branch update: %w", err)
		}

		pr, err = s.pullreqStore.UpdateActivitySeq(ctx, pr)
		if err != nil {
			return fmt.Errorf("failed to get pull request activity number after branch update: %w", err)
		}

		now := time.Now().UnixMilli()
		act := &types.PullReqActivity{
			CreatedBy: event.Payload.PrincipalID,
			Created:   now,
			Updated:   now,
			Edited:    now,
			RepoID:    pr.TargetRepoID,
			PullReqID: pr.ID,
			Order:     pr.ActivitySeq,
			SubOrder:  0,
			ReplySeq:  0,
			Type:      enum.PullReqActivityTypeBranchUpdate,
			Kind:      enum.PullReqActivityKindSystem,
			Text:      "",
		}

		_ = act.SetPayload(&types.PullRequestActivityPayloadBranchUpdate{
			Old: event.Payload.OldSHA,
			New: event.Payload.NewSHA,
		})

		err = s.activityStore.Create(ctx, act)
		if err != nil {
			return fmt.Errorf("failed to create pull request activity: %w", err)
		}

		return nil
	})
	return nil
}

func (s *Service) handleEventBranchDeleted(ctx context.Context,
	event *events.Event[*gitevents.BranchDeletedPayload],
) error {
	s.forEveryOpenPR(ctx, event.Payload.RepoID, event.Payload.Ref, func(pr *types.PullReq) error {
		pr, err := s.pullreqStore.UpdateOptLock(ctx, pr, func(pr *types.PullReq) error {
			pr.ActivitySeq++
			pr.State = enum.PullReqStateClosed
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to close and get pull request activity number after branch delete: %w", err)
		}

		now := time.Now().UnixMilli()
		act := &types.PullReqActivity{
			CreatedBy: event.Payload.PrincipalID,
			Created:   now,
			Updated:   now,
			Edited:    now,
			RepoID:    pr.TargetRepoID,
			PullReqID: pr.ID,
			Order:     pr.ActivitySeq,
			SubOrder:  0,
			ReplySeq:  0,
			Type:      enum.PullReqActivityTypeBranchDelete,
			Kind:      enum.PullReqActivityKindSystem,
			Text:      "",
		}

		_ = act.SetPayload(&types.PullRequestActivityPayloadBranchDelete{
			SHA: event.Payload.SHA,
		})

		err = s.activityStore.Create(ctx, act)
		if err != nil {
			return fmt.Errorf("failed to create pull request activity: %w", err)
		}

		return nil
	})
	return nil
}

func (s *Service) forEveryOpenPR(ctx context.Context,
	repoID int64, ref string,
	fn func(pr *types.PullReq) error,
) {
	const refPrefix = "refs/heads/"
	const largeLimit = 1000000

	if !strings.HasPrefix(ref, refPrefix) {
		log.Ctx(ctx).Error().Msg("failed to get branch name from branch ref")
		return
	}

	branch := ref[len(refPrefix):]
	if len(branch) == 0 {
		log.Ctx(ctx).Error().Msg("got an empty branch name from branch ref")
		return
	}

	pullreqList, err := s.pullreqStore.List(ctx, &types.PullReqFilter{
		Page:         0,
		Size:         largeLimit,
		SourceRepoID: repoID,
		SourceBranch: branch,
		States:       []enum.PullReqState{enum.PullReqStateOpen},
		Sort:         enum.PullReqSortNumber,
		Order:        enum.OrderAsc,
	})
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to get list of open pull requests")
		return
	}

	for _, pr := range pullreqList {
		if err = fn(pr); err != nil {
			log.Ctx(ctx).Err(err).Msg("failed to process pull req")
		}
	}
}
