// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package scheduler

import (
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/lock"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideScheduler,
)

// ProvideScheduler provides a scheduler which can be used to schedule and request builds.
func ProvideScheduler(
	stageStore store.StageStore,
	lock lock.MutexManager,
) (Scheduler, error) {
	return newScheduler(stageStore, lock)
}
