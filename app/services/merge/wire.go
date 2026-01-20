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
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideService,
)

func ProvideService(ctx context.Context,
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
	return NewService(
		ctx,
		config,
		git,
		eventReporter,
		statusCheckFactory,
		pullreqEvReaderFactory,
		repoFinder,
		repoStore,
		pullreqStore,
		activityStore,
		checkStore,
		reviewerStore,
		principalInfoCache,
		principalStore,
		autoMergeStore,
		protectionManager,
		codeOwners,
		userGroupService,
		urlProvider,
		sseStreamer,
		pubsubBus,
		instrumentation,
		locker,
	)
}
