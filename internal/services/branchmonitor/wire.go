// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package branchmonitor

import (
	"context"

	"github.com/harness/gitness/events"
	gitevents "github.com/harness/gitness/internal/events/git"
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
	db *sqlx.DB,
	pullreqStore store.PullReqStore,
	activityStore store.PullReqActivityStore,
) (*Service, error) {
	return New(ctx, config, gitReaderFactory, db, pullreqStore, activityStore)
}
