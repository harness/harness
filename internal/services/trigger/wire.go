// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package trigger

import (
	"context"

	"github.com/harness/gitness/events"
	gitevents "github.com/harness/gitness/internal/events/git"
	pullreqevents "github.com/harness/gitness/internal/events/pullreq"

	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideService,
)

func ProvideService(
	ctx context.Context,
	config Config,
	gitReaderFactory *events.ReaderFactory[*gitevents.Reader],
	pullReqEvFactory *events.ReaderFactory[*pullreqevents.Reader],
) (*Service, error) {
	return New(ctx, config, gitReaderFactory, pullReqEvFactory)
}
