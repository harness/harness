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
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideService,
)

func ProvideService(
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
	return NewService(
		ctx,
		config,
		git,
		tx,
		mergeQueueEventReporter,
		statusCheckFactory,
		mergeQueueEvReaderFactory,
		repoFinder,
		repoStore,
		pullreqStore,
		activityStore,
		checkStore,
		mergeQueueStore,
		mergeQueueEntryStore,
		protectionManager,
		mergeService,
		urlProvider,
		locker,
	)
}
