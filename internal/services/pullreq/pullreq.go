// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"
	"sync"
	"time"

	"github.com/harness/gitness/events"
	"github.com/harness/gitness/gitrpc"
	gitevents "github.com/harness/gitness/internal/events/git"
	pullreqevents "github.com/harness/gitness/internal/events/pullreq"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/stream"
	"github.com/harness/gitness/types"

	"github.com/jmoiron/sqlx"
)

type Service struct {
	pullreqEvReporter *pullreqevents.Reporter
	gitRPCClient      gitrpc.Interface
	db                *sqlx.DB
	repoGitInfoCache  store.RepoGitInfoCache
	principalCache    store.PrincipalInfoCache
	repoStore         store.RepoStore
	pullreqStore      store.PullReqStore
	activityStore     store.PullReqActivityStore

	cancelMutex       sync.Mutex
	cancelMergability map[string]context.CancelFunc

	pubsub pubsub.PubSub
}

//nolint:funlen // needs refactoring
func New(ctx context.Context,
	config *types.Config,
	gitReaderFactory *events.ReaderFactory[*gitevents.Reader],
	pullreqEvReaderFactory *events.ReaderFactory[*pullreqevents.Reader],
	pullreqEvReporter *pullreqevents.Reporter,
	gitRPCClient gitrpc.Interface,
	db *sqlx.DB,
	repoGitInfoCache store.RepoGitInfoCache,
	principalCache store.PrincipalInfoCache,
	repoStore store.RepoStore,
	pullreqStore store.PullReqStore,
	activityStore store.PullReqActivityStore,
	bus pubsub.PubSub,
) (*Service, error) {
	service := &Service{
		pullreqEvReporter: pullreqEvReporter,
		gitRPCClient:      gitRPCClient,
		db:                db,
		repoGitInfoCache:  repoGitInfoCache,
		principalCache:    principalCache,
		repoStore:         repoStore,
		pullreqStore:      pullreqStore,
		activityStore:     activityStore,
		cancelMergability: make(map[string]context.CancelFunc),
		pubsub:            bus,
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

	// mergability check
	const groupPullReqMergeable = "gitness:pullreq:mergeable"
	_, err = pullreqEvReaderFactory.Launch(ctx, groupPullReqMergeable, config.InstanceID,
		func(r *pullreqevents.Reader) error {
			const idleTimeout = 30 * time.Second
			r.Configure(
				stream.WithConcurrency(3),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(idleTimeout),
					stream.WithMaxRetries(2),
				))

			_ = r.RegisterCreated(service.mergeCheckOnCreated)
			_ = r.RegisterBranchUpdated(service.mergeCheckOnBranchUpdate)
			_ = r.RegisterReopened(service.mergeCheckOnReopen)
			_ = r.RegisterClosed(service.mergeCheckOnClosed)
			_ = r.RegisterMerged(service.mergeCheckOnMerged)

			return nil
		})
	if err != nil {
		return nil, err
	}

	// cancel any previous pr mergability check
	// payload is oldsha.
	_ = bus.Subscribe(ctx, cancelMergeCheckKey, func(payload []byte) error {
		oldSHA := string(payload)
		if oldSHA == "" {
			return nil
		}

		service.cancelMutex.Lock()
		defer service.cancelMutex.Unlock()

		cancel := service.cancelMergability[oldSHA]
		if cancel != nil {
			cancel()
		}

		delete(service.cancelMergability, oldSHA)

		return nil
	}, pubsub.WithChannelNamespace("pullreq"))

	return service, nil
}
