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
	"github.com/harness/gitness/stream"
	"github.com/harness/gitness/types"

	"github.com/jmoiron/sqlx"
)

type Service struct {
	pullreqEvReporter *pullreqevents.Reporter
	gitRPCClient      gitrpc.Interface
	db                *sqlx.DB
	repoGitInfoCache  store.RepoGitInfoCache
	repoStore         store.RepoStore
	pullreqStore      store.PullReqStore
	activityStore     store.PullReqActivityStore
}

func New(ctx context.Context,
	config *types.Config,
	gitReaderFactory *events.ReaderFactory[*gitevents.Reader],
	pullreqEvReaderFactory *events.ReaderFactory[*pullreqevents.Reader],
	pullreqEvReporter *pullreqevents.Reporter,
	gitRPCClient gitrpc.Interface,
	db *sqlx.DB,
	repoGitInfoCache store.RepoGitInfoCache,
	repoStore store.RepoStore,
	pullreqStore store.PullReqStore,
	activityStore store.PullReqActivityStore,
) (*Service, error) {
	service := &Service{
		pullreqEvReporter: pullreqEvReporter,
		gitRPCClient:      gitRPCClient,
		db:                db,
		repoGitInfoCache:  repoGitInfoCache,
		repoStore:         repoStore,
		pullreqStore:      pullreqStore,
		activityStore:     activityStore,
	}

	var err error

	// handle git branch events to trigger specific pull request events

	const groupGit = "gitness:pullreq:git"
	_, err = gitReaderFactory.Launch(ctx, groupGit, config.InstanceID,
		func(r *gitevents.Reader) error {
			const idleTimeout = 15 * time.Second
			r.Configure(
				stream.WithConcurrency(1),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(idleTimeout),
					stream.WithMaxRetries(3),
				))

			_ = r.RegisterBranchUpdated(service.triggerPullReqBranchUpdate)
			_ = r.RegisterBranchDeleted(service.closePullReqBranchDelete)

			return nil
		})
	if err != nil {
		return nil, err
	}

	// pull request timeline activity generation

	const groupActivity = "gitness:pullreq:activity"
	_, err = pullreqEvReaderFactory.Launch(ctx, groupActivity, config.InstanceID,
		func(r *pullreqevents.Reader) error {
			const idleTimeout = 10 * time.Second
			r.Configure(
				stream.WithConcurrency(1),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(idleTimeout),
					stream.WithMaxRetries(3),
				))
			_ = r.RegisterBranchUpdated(service.addActivityBranchUpdate)
			_ = r.RegisterBranchDeleted(service.addActivityBranchDelete)
			_ = r.RegisterStateChanged(service.addActivityStateChange)
			_ = r.RegisterTitleChanged(service.addActivityTitleChange)
			_ = r.RegisterReviewSubmitted(service.addActivityReviewSubmit)
			_ = r.RegisterMerged(service.addActivityMerge)

			return nil
		})
	if err != nil {
		return nil, err
	}

	// pull request ref maintenance

	const groupPullReqHeadRef = "gitness:pullreq:headref"
	_, err = pullreqEvReaderFactory.Launch(ctx, groupPullReqHeadRef, config.InstanceID,
		func(r *pullreqevents.Reader) error {
			const idleTimeout = 10 * time.Second
			r.Configure(
				stream.WithConcurrency(1),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(idleTimeout),
					stream.WithMaxRetries(3),
				))

			_ = r.RegisterCreated(service.createHeadRefCreated)
			_ = r.RegisterBranchUpdated(service.updateHeadRefBranchUpdate)
			_ = r.RegisterStateChanged(service.updateHeadRefStateChange)

			return nil
		})
	if err != nil {
		return nil, err
	}

	return service, nil
}
