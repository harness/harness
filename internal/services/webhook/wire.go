// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package webhook

import (
	"context"

	"github.com/harness/gitness/events"
	"github.com/harness/gitness/gitrpc"
	gitevents "github.com/harness/gitness/internal/events/git"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/url"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideService,
)

func ProvideService(ctx context.Context, config Config,
	gitReaderFactory *events.ReaderFactory[*gitevents.Reader],
	webhookStore store.WebhookStore, webhookExecutionStore store.WebhookExecutionStore,
	repoStore store.RepoStore, urlProvider *url.Provider,
	principalStore store.PrincipalStore, gitRPCClient gitrpc.Interface) (*Service, error) {
	return NewService(ctx, config, gitReaderFactory, webhookStore, webhookExecutionStore,
		repoStore, urlProvider, principalStore, gitRPCClient)
}
