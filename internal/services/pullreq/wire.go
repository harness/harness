// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"

	"github.com/harness/gitness/events"
	"github.com/harness/gitness/gitrpc"
	gitevents "github.com/harness/gitness/internal/events/git"
	pullreqevents "github.com/harness/gitness/internal/events/pullreq"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
	"github.com/jmoiron/sqlx"
)

var WireSet = wire.NewSet(
	ProvideService,
)

func ProvideService(ctx context.Context,
	config *types.Config,
	gitReaderFactory *events.ReaderFactory[*gitevents.Reader],
	pullReqEventFactory *events.ReaderFactory[*pullreqevents.Reader],
	gitRPCClient gitrpc.Interface,
	db *sqlx.DB,
	repoStore store.RepoStore,
	pullreqStore store.PullReqStore,
	activityStore store.PullReqActivityStore,
) (*Service, error) {
	return New(ctx, config, gitReaderFactory, pullReqEventFactory, gitRPCClient,
		db, repoStore, pullreqStore, activityStore)
}
