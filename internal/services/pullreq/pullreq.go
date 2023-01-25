// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"
	"time"

	"github.com/harness/gitness/events"
	"github.com/harness/gitness/gitrpc"
	gitevents "github.com/harness/gitness/internal/events/git"
	pullreqevents "github.com/harness/gitness/internal/events/pullreq"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"

	"github.com/jmoiron/sqlx"
)

const (
	eventsReaderGroupName = "pullreq"
)

type Service struct {
	gitRPCClient  gitrpc.Interface
	db            *sqlx.DB
	repoStore     store.RepoStore
	pullreqStore  store.PullReqStore
	activityStore store.PullReqActivityStore
}

func New(ctx context.Context,
	config *types.Config,
	gitReaderFactory *events.ReaderFactory[*gitevents.Reader],
	pullreqEvReaderFactory *events.ReaderFactory[*pullreqevents.Reader],
	gitRPCClient gitrpc.Interface,
	db *sqlx.DB,
	repoStore store.RepoStore,
	pullreqStore store.PullReqStore,
	activityStore store.PullReqActivityStore,
) (*Service, error) {
	service := &Service{
		gitRPCClient:  gitRPCClient,
		db:            db,
		repoStore:     repoStore,
		pullreqStore:  pullreqStore,
		activityStore: activityStore,
	}

	var err error

	_, err = gitReaderFactory.Launch(ctx, eventsReaderGroupName, config.InstanceID,
		func(r *gitevents.Reader) error {
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

	_, err = pullreqEvReaderFactory.Launch(ctx, eventsReaderGroupName, config.InstanceID,
		func(r *pullreqevents.Reader) error {
			const processingTimeout = 30 * time.Second

			_ = r.SetConcurrency(1)
			_ = r.SetMaxRetryCount(1)
			_ = r.SetProcessingTimeout(processingTimeout)

			_ = r.RegisterCreated(service.handleEventPullReqCreated)
			_ = r.RegisterUpdated(service.handleEventPullReqUpdated)
			_ = r.RegisterStateChange(service.handleEventPullReqStateChange)
			_ = r.RegisterMerged(service.handleEventPullReqMerged)

			return nil
		})
	if err != nil {
		return nil, err
	}

	return service, nil
}
