// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package webhook

import (
	"context"

	"github.com/harness/gitness/events"
	gitevents "github.com/harness/gitness/gitrpc/events"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideServer,
)

func ProvideServer(ctx context.Context, config *types.Config,
	gitReaderFactory *events.ReaderFactory[*gitevents.Reader]) (*Server, error) {
	// Use instanceID as readerName as every instance should be one reader
	readerName := config.InstanceID
	return NewServer(ctx, gitReaderFactory, readerName, config.Webhook.Concurrency)
}
