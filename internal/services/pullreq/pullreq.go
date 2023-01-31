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

			_ = r.RegisterBranchUpdated(service.triggerPREventOnBranchUpdate)
			_ = r.RegisterBranchDeleted(service.closePullReqOnBranchDelete)

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

			_ = r.RegisterCreated(service.createHeadRefOnCreated)
			_ = r.RegisterBranchUpdated(service.updateHeadRefOnBranchUpdate)
			_ = r.RegisterReopened(service.updateHeadRefOnReopen)

			return nil
		})
	if err != nil {
		return nil, err
	}

	const groupPullReqCounters = "gitness:pullreq:counters"
	_, err = pullreqEvReaderFactory.Launch(ctx, groupPullReqCounters, config.InstanceID,
		func(r *pullreqevents.Reader) error {
			const idleTimeout = 10 * time.Second
			r.Configure(
				stream.WithConcurrency(1),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(idleTimeout),
					stream.WithMaxRetries(2),
				))

			_ = r.RegisterCreated(service.updatePRCountersOnCreated)
			_ = r.RegisterReopened(service.updatePRCountersOnReopened)
			_ = r.RegisterClosed(service.updatePRCountersOnClosed)
			_ = r.RegisterMerged(service.updatePRCountersOnMerged)

			return nil
		})
	if err != nil {
		return nil, err
	}

	return service, nil
}
