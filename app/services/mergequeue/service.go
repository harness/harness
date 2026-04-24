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

package mergequeue

import (
	"context"
	"time"

	checkevents "github.com/harness/gitness/app/events/check"
	mergequeueevents "github.com/harness/gitness/app/events/mergequeue"
	"github.com/harness/gitness/app/services/locker"
	"github.com/harness/gitness/app/services/merge"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/stream"
	"github.com/harness/gitness/types"
)

const (
	MaximumQueueSize = 100
)

type Service struct {
	git                       git.Interface
	tx                        dbtx.Transactor
	mergeQueueEventReporter   *mergequeueevents.Reporter
	mergeQueueEvReaderFactory *events.ReaderFactory[*mergequeueevents.Reader]
	repoFinder                refcache.RepoFinder
	repoStore                 store.RepoStore
	pullreqStore              store.PullReqStore
	activityStore             store.PullReqActivityStore
	checkStore                store.CheckStore
	mergeQueueStore           store.MergeQueueStore
	mergeQueueEntryStore      store.MergeQueueEntryStore
	protectionManager         *protection.Manager
	mergeService              *merge.Service
	urlProvider               url.Provider
	locker                    *locker.Locker
}

func NewService(
	ctx context.Context,
	config *types.Config,
	git git.Interface,
	tx dbtx.Transactor,
	mergeQueueEventReporter *mergequeueevents.Reporter,
	statusCheckFactory *events.ReaderFactory[*checkevents.Reader],
	mergeQueueEvReaderFactory *events.ReaderFactory[*mergequeueevents.Reader],
	repoFinder refcache.RepoFinder,
	repoStore store.RepoStore,
	pullreqStore store.PullReqStore,
	activityStore store.PullReqActivityStore,
	checkStore store.CheckStore,
	mergeQueueStore store.MergeQueueStore,
	mergeQueueEntryStore store.MergeQueueEntryStore,
	protectionManager *protection.Manager,
	mergeService *merge.Service,
	urlProvider url.Provider,
	locker *locker.Locker,
) (*Service, error) {
	service := &Service{
		git:                       git,
		tx:                        tx,
		mergeQueueEventReporter:   mergeQueueEventReporter,
		mergeQueueEvReaderFactory: mergeQueueEvReaderFactory,
		repoFinder:                repoFinder,
		repoStore:                 repoStore,
		pullreqStore:              pullreqStore,
		activityStore:             activityStore,
		checkStore:                checkStore,
		mergeQueueStore:           mergeQueueStore,
		mergeQueueEntryStore:      mergeQueueEntryStore,
		protectionManager:         protectionManager,
		mergeService:              mergeService,
		urlProvider:               urlProvider,
		locker:                    locker,
	}

	var err error

	const groupMergeQueue = "gitness:merge_queue"

	_, err = statusCheckFactory.Launch(ctx, groupMergeQueue, config.InstanceID,
		func(r *checkevents.Reader) error {
			const idleTimeout = 15 * time.Second
			r.Configure(
				stream.WithConcurrency(1),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(idleTimeout),
					stream.WithMaxRetries(3),
				))

			_ = r.RegisterReported(service.handlerCheckFinished)

			return nil
		})
	if err != nil {
		return nil, err
	}

	_, err = mergeQueueEvReaderFactory.Launch(ctx, groupMergeQueue, config.InstanceID,
		func(r *mergequeueevents.Reader) error {
			const idleTimeout = 15 * time.Second
			r.Configure(
				stream.WithConcurrency(1),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(idleTimeout),
					stream.WithMaxRetries(3),
				))

			_ = r.RegisterUpdated(service.handlerUpdated)

			return nil
		})
	if err != nil {
		return nil, err
	}

	return service, nil
}
