// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
