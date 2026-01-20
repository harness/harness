// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package merge

import (
	"context"
	"time"

	checkevents "github.com/harness/gitness/app/events/check"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/services/codeowners"
	"github.com/harness/gitness/app/services/instrument"
	"github.com/harness/gitness/app/services/locker"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/services/usergroup"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/stream"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type Service struct {
	git                git.Interface
	eventReporter      *pullreqevents.Reporter
	repoFinder         refcache.RepoFinder
	repoStore          store.RepoStore
	pullreqStore       store.PullReqStore
	activityStore      store.PullReqActivityStore
	checkStore         store.CheckStore
	reviewerStore      store.PullReqReviewerStore
	principalInfoCache store.PrincipalInfoCache
	principalStore     store.PrincipalStore
	autoMergeStore     store.AutoMergeStore
	protectionManager  *protection.Manager
	codeOwners         *codeowners.Service
	userGroupService   usergroup.Service
	urlProvider        url.Provider
	sseStreamer        sse.Streamer
	pubsub             pubsub.PubSub
	instrumentation    instrument.Service
	locker             *locker.Locker
}

func NewService(
	ctx context.Context,
	config *types.Config,
	git git.Interface,
	eventReporter *pullreqevents.Reporter,
	statusCheckFactory *events.ReaderFactory[*checkevents.Reader],
	pullreqEvReaderFactory *events.ReaderFactory[*pullreqevents.Reader],
	repoFinder refcache.RepoFinder,
	repoStore store.RepoStore,
	pullreqStore store.PullReqStore,
	activityStore store.PullReqActivityStore,
	checkStore store.CheckStore,
	reviewerStore store.PullReqReviewerStore,
	principalInfoCache store.PrincipalInfoCache,
	principalStore store.PrincipalStore,
	autoMergeStore store.AutoMergeStore,
	protectionManager *protection.Manager,
	codeOwners *codeowners.Service,
	userGroupService usergroup.Service,
	urlProvider url.Provider,
	sseStreamer sse.Streamer,
	pubsubBus pubsub.PubSub,
	instrumentation instrument.Service,
	locker *locker.Locker,
) (*Service, error) {
	service := &Service{
		git:                git,
		eventReporter:      eventReporter,
		repoFinder:         repoFinder,
		repoStore:          repoStore,
		pullreqStore:       pullreqStore,
		activityStore:      activityStore,
		checkStore:         checkStore,
		reviewerStore:      reviewerStore,
		principalInfoCache: principalInfoCache,
		principalStore:     principalStore,
		autoMergeStore:     autoMergeStore,
		protectionManager:  protectionManager,
		codeOwners:         codeOwners,
		userGroupService:   userGroupService,
		urlProvider:        urlProvider,
		sseStreamer:        sseStreamer,
		pubsub:             pubsubBus,
		instrumentation:    instrumentation,
		locker:             locker,
	}

	var err error

	const groupAutoMerge = "gitness:automerge"

	_, err = statusCheckFactory.Launch(ctx, groupAutoMerge, config.InstanceID,
		func(r *checkevents.Reader) error {
			const idleTimeout = 15 * time.Second
			r.Configure(
				stream.WithConcurrency(1),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(idleTimeout),
					stream.WithMaxRetries(3),
				))

			_ = r.RegisterReported(service.mergePRsOnCheckSucceeded)

			return nil
		})
	if err != nil {
		return nil, err
	}

	_, err = pullreqEvReaderFactory.Launch(ctx, groupAutoMerge, config.InstanceID,
		func(r *pullreqevents.Reader) error {
			const idleTimeout = 30 * time.Second
			r.Configure(
				stream.WithConcurrency(3),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(idleTimeout),
					stream.WithMaxRetries(2),
				))

			_ = r.RegisterMergeCheckSucceeded(service.mergePRsOnMergeCheckSucceeded)
			_ = r.RegisterReviewSubmitted(service.mergePRsOnApproval)
			_ = r.RegisterCommentStatusUpdated(service.mergePRsOnCommentResolve)

			return nil
		})
	if err != nil {
		return nil, err
	}

	return service, nil
}

func isEligibleForAutoMerge(pr *types.PullReq) bool {
	return pr.State == enum.PullReqStateOpen &&
		pr.SubState == enum.PullReqSubStateAutoMerge &&
		!pr.IsDraft &&
		pr.MergeCheckStatus == enum.MergeCheckStatusMergeable
}
