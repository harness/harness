// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package triggerer

import (
	"github.com/harness/gitness/internal/pipeline/file"
	"github.com/harness/gitness/internal/pipeline/scheduler"
	"github.com/harness/gitness/internal/store"

	"github.com/google/wire"
	"github.com/jmoiron/sqlx"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideTriggerer,
)

// ProvideTriggerer provides a triggerer which can execute builds.
func ProvideTriggerer(
	executionStore store.ExecutionStore,
	checkStore store.CheckStore,
	stageStore store.StageStore,
	db *sqlx.DB,
	pipelineStore store.PipelineStore,
	fileService file.FileService,
	scheduler scheduler.Scheduler,
	repoStore store.RepoStore,
) Triggerer {
	return New(executionStore, checkStore, stageStore, pipelineStore,
		db, repoStore, scheduler, fileService)
}
