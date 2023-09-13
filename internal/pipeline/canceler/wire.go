// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package canceler

import (
	"github.com/harness/gitness/internal/pipeline/scheduler"
	"github.com/harness/gitness/internal/sse"
	"github.com/harness/gitness/internal/store"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideCanceler,
)

// ProvideExecutionManager provides an execution manager.
func ProvideCanceler(
	executionStore store.ExecutionStore,
	sseStreamer sse.Streamer,
	repoStore store.RepoStore,
	scheduler scheduler.Scheduler,
	stageStore store.StageStore,
	stepStore store.StepStore) Canceler {
	return New(executionStore, sseStreamer, repoStore, scheduler, stageStore, stepStore)
}
