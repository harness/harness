// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/harness/gitness/events"
	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/bootstrap"
	gitevents "github.com/harness/gitness/internal/events/git"
	pullreqevents "github.com/harness/gitness/internal/events/pullreq"
	"github.com/harness/gitness/internal/githook"
	"github.com/harness/gitness/internal/services/codecomments"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/url"
	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/stream"
	"github.com/harness/gitness/types"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

type Service struct {
	pullreqEvReporter   *pullreqevents.Reporter
	gitRPCClient        gitrpc.Interface
	db                  *sqlx.DB
	repoGitInfoCache    store.RepoGitInfoCache
	repoStore           store.RepoStore
	pullreqStore        store.PullReqStore
	activityStore       store.PullReqActivityStore
	codeCommentView     store.CodeCommentView
	codeCommentMigrator *codecomments.Migrator
	urlProvider         *url.Provider

	cancelMutex        sync.Mutex
	cancelMergeability map[string]context.CancelFunc

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
	repoStore store.RepoStore,
	pullreqStore store.PullReqStore,
	activityStore store.PullReqActivityStore,
	codeCommentView store.CodeCommentView,
	codeCommentMigrator *codecomments.Migrator,
	bus pubsub.PubSub,
	urlProvider *url.Provider,
) (*Service, error) {
	service := &Service{
		pullreqEvReporter:   pullreqEvReporter,
		gitRPCClient:        gitRPCClient,
		db:                  db,
		repoGitInfoCache:    repoGitInfoCache,
		repoStore:           repoStore,
		pullreqStore:        pullreqStore,
		activityStore:       activityStore,
		codeCommentView:     codeCommentView,
		urlProvider:         urlProvider,
		codeCommentMigrator: codeCommentMigrator,
		cancelMergeability:  make(map[string]context.CancelFunc),
		pubsub:              bus,
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

	// mergeability check
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

	// cancel any previous pr mergeability check
	// payload is oldsha.
	_ = bus.Subscribe(ctx, cancelMergeCheckKey, func(payload []byte) error {
		oldSHA := string(payload)
		if oldSHA == "" {
			return nil
		}

		service.cancelMutex.Lock()
		defer service.cancelMutex.Unlock()

		cancel := service.cancelMergeability[oldSHA]
		if cancel != nil {
			cancel()
		}

		delete(service.cancelMergeability, oldSHA)

		return nil
	}, pubsub.WithChannelNamespace("pullreq"))

	// mergeability check
	const groupPullReqCodeComments = "gitness:pullreq:codecomments"
	_, err = pullreqEvReaderFactory.Launch(ctx, groupPullReqCodeComments, config.InstanceID,
		func(r *pullreqevents.Reader) error {
			const idleTimeout = 10 * time.Second
			r.Configure(
				stream.WithConcurrency(3),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(idleTimeout),
					stream.WithMaxRetries(2),
				))

			_ = r.RegisterBranchUpdated(service.updateCodeCommentsOnBranchUpdate)
			_ = r.RegisterReopened(service.updateCodeCommentsOnReopen)

			return nil
		})
	if err != nil {
		return nil, err
	}

	return service, nil
}

// createSystemRPCWriteParams creates base write parameters for gitrpc write operations.
func createSystemRPCWriteParams(ctx context.Context, urlProvider *url.Provider,
	repoID int64, repoGITUID string) (gitrpc.WriteParams, error) {
	requestID, ok := request.RequestIDFrom(ctx)
	if !ok {
		// best effort retrieving of requestID - log in case we can't find it but don't fail operation.
		log.Ctx(ctx).Warn().Msg("operation doesn't have a requestID in the context.")
	}

	principal := bootstrap.NewSystemServiceSession().Principal

	// generate envars (add everything githook CLI needs for execution)
	envVars, err := githook.GenerateEnvironmentVariables(&githook.Payload{
		APIBaseURL:  urlProvider.GetAPIBaseURLInternal(),
		RepoID:      repoID,
		PrincipalID: principal.ID,
		RequestID:   requestID,
		Disabled:    false,
	})
	if err != nil {
		return gitrpc.WriteParams{}, fmt.Errorf("failed to generate git hook environment variables: %w", err)
	}

	return gitrpc.WriteParams{
		Actor: gitrpc.Identity{
			Name:  principal.DisplayName,
			Email: principal.Email,
		},
		RepoUID: repoGITUID,
		EnvVars: envVars,
	}, nil
}
