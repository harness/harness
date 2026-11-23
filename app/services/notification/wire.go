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

package notification

import (
	"context"

	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/services/notification/mailer"
	"github.com/harness/gitness/app/services/usergroup"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/idempotency"
	"github.com/harness/gitness/types"

	"github.com/go-redis/redis/v8"
	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideMailClient,
	ProvideNotificationService,
)

func ProvideNotificationService(
	ctx context.Context,
	notificationClient Client,
	pullReqConfig Config,
	prReaderFactory *events.ReaderFactory[*pullreqevents.Reader],
	pullReqStore store.PullReqStore,
	repoStore store.RepoStore,
	principalInfoView store.PrincipalInfoView,
	principalInfoCache store.PrincipalInfoCache,
	pullReqReviewersStore store.PullReqReviewerStore,
	pullReqWatcherStore store.PullReqWatcherStore,
	userGroupReviewerStore store.UserGroupReviewerStore,
	userGroupService usergroup.Service,
	pullReqActivityStore store.PullReqActivityStore,
	spacePathStore store.SpacePathStore,
	urlProvider url.Provider,
) (*Service, error) {
	return NewService(
		ctx,
		pullReqConfig,
		notificationClient,
		prReaderFactory,
		pullReqStore,
		repoStore,
		principalInfoView,
		principalInfoCache,
		pullReqReviewersStore,
		pullReqWatcherStore,
		userGroupReviewerStore,
		userGroupService,
		pullReqActivityStore,
		spacePathStore,
		urlProvider,
	)
}

// ProvideMailClient wraps the mailer to deduplicate mails of redelivered events.
func ProvideMailClient(config *types.Config, m mailer.Mailer, redisClient redis.UniversalClient) Client {
	var store idempotency.Store = idempotency.NewInMemory()
	if config.Events.Mode == events.ModeRedis {
		store = idempotency.NewRedis(redisClient)
	}

	return NewMailClient(mailer.NewDedupMailer(m, store, mailer.DefaultDedupTTL))
}
