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

package webhook

import (
	"context"

	gitevents "github.com/harness/gitness/app/events/git"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/encrypt"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideService,
	ProvideURLProvider,
)

func ProvideService(
	ctx context.Context,
	config Config,
	tx dbtx.Transactor,
	gitReaderFactory *events.ReaderFactory[*gitevents.Reader],
	prReaderFactory *events.ReaderFactory[*pullreqevents.Reader],
	webhookStore store.WebhookStore,
	webhookExecutionStore store.WebhookExecutionStore,
	spaceStore store.SpaceStore,
	repoStore store.RepoStore,
	pullreqStore store.PullReqStore,
	activityStore store.PullReqActivityStore,
	urlProvider url.Provider,
	principalStore store.PrincipalStore,
	git git.Interface,
	encrypter encrypt.Encrypter,
	labelStore store.LabelStore,
	webhookURLProvider URLProvider,
	labelValueStore store.LabelValueStore,
	sseStreamer sse.Streamer,
) (*Service, error) {
	return NewService(
		ctx,
		config,
		tx,
		gitReaderFactory,
		prReaderFactory,
		webhookStore,
		webhookExecutionStore,
		spaceStore, repoStore,
		pullreqStore,
		activityStore,
		urlProvider,
		principalStore,
		git,
		encrypter,
		labelStore,
		webhookURLProvider,
		labelValueStore,
		sseStreamer,
	)
}

func ProvideURLProvider(ctx context.Context) URLProvider {
	return NewURLProvider(ctx)
}
