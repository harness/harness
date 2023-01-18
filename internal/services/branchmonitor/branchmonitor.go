// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package branchmonitor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/harness/gitness/events"
	gitevents "github.com/harness/gitness/internal/events/git"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

const (
	eventsReaderGroupName = "pullreq"
)

type Service struct {
	db            *sqlx.DB
	pullreqStore  store.PullReqStore
	activityStore store.PullReqActivityStore
}

const largeLimit = 1000000

func New(ctx context.Context,
	config *types.Config,
	gitReaderFactory *events.ReaderFactory[*gitevents.Reader],
	db *sqlx.DB,
	pullreqStore store.PullReqStore,
	activityStore store.PullReqActivityStore,
) (*Service, error) {
	service := &Service{
		db:            db,
		pullreqStore:  pullreqStore,
		activityStore: activityStore,
	}

	_, err := gitReaderFactory.Launch(ctx, eventsReaderGroupName, config.InstanceID, func(r *gitevents.Reader) error {
		const processingTimeout = 15 * time.Second

		_ = r.SetConcurrency(1)
		_ = r.SetMaxRetryCount(1)
		_ = r.SetProcessingTimeout(processingTimeout)

		_ = r.RegisterBranchUpdated(service.handleEventBranchUpdated)
		_ = r.RegisterBranchDeleted(service.handleEventBranchDeleted)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return service, nil
}

func (s *Service) handleEventBranchUpdated(ctx context.Context,
	event *events.Event[*gitevents.BranchUpdatedPayload],
) error {
	s.forEveryOpenPR(ctx, event.Payload.RepoID, event.Payload.Ref, func(pr *types.PullReq) error {
		pr, err := s.pullreqStore.UpdateActivitySeq(ctx, pr)
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
			Payload: map[string]interface{}{
				"old": event.Payload.OldSHA,
				"new": event.Payload.NewSHA,
			},
		}

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
			Payload: map[string]interface{}{
				"sha": event.Payload.SHA,
			},
		}

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
	if !strings.HasPrefix(ref, refPrefix) {
		log.Error().Msg("failed to get branch name from branch ref")
		return
	}

	branch := ref[len(refPrefix):]
	if len(branch) == 0 {
		log.Error().Msg("got an empty branch name from branch ref")
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
		log.Err(err).Msg("failed to get list of open pull requests")
		return
	}

	for _, pr := range pullreqList {
		if err = fn(pr); err != nil {
			log.Err(err).Msg("failed to process pull req")
		}
	}
}
