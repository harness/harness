// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package job

import (
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/lock"
	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideExecutor,
	ProvideScheduler,
)

func ProvideExecutor(
	jobStore store.JobStore,
	pubsubService pubsub.PubSub,
) *Executor {
	return NewExecutor(
		jobStore,
		pubsubService,
	)
}

func ProvideScheduler(
	jobStore store.JobStore,
	executor *Executor,
	mutexManager lock.MutexManager,
	pubsubService pubsub.PubSub,
	config *types.Config,
) (*Scheduler, error) {
	return NewScheduler(
		jobStore,
		executor,
		mutexManager,
		pubsubService,
		config.InstanceID,
		config.BackgroundJobs.MaxRunning,
		config.BackgroundJobs.PurgeFinishedOlderThan,
	)
}
