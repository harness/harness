// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package trigger

import (
	"context"

	"github.com/harness/gitness/events"
	gitevents "github.com/harness/gitness/internal/events/git"
	pullreqevents "github.com/harness/gitness/internal/events/pullreq"
	"github.com/harness/gitness/internal/pipeline/commit"
	"github.com/harness/gitness/internal/pipeline/triggerer"
	"github.com/harness/gitness/internal/store"

	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideService,
)

func ProvideService(
	ctx context.Context,
	config Config,
	triggerStore store.TriggerStore,
	commitSvc commit.CommitService,
	pullReqStore store.PullReqStore,
	repoStore store.RepoStore,
	pipelineStore store.PipelineStore,
	triggerSvc triggerer.Triggerer,
	gitReaderFactory *events.ReaderFactory[*gitevents.Reader],
	pullReqEvFactory *events.ReaderFactory[*pullreqevents.Reader],
) (*Service, error) {
	return New(ctx, config, triggerStore, pullReqStore, repoStore, pipelineStore, triggerSvc,
		commitSvc, gitReaderFactory, pullReqEvFactory)
}
