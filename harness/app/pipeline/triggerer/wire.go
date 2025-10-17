// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package triggerer

import (
	"github.com/harness/gitness/app/pipeline/converter"
	"github.com/harness/gitness/app/pipeline/file"
	"github.com/harness/gitness/app/pipeline/scheduler"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/google/wire"
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
	tx dbtx.Transactor,
	pipelineStore store.PipelineStore,
	fileService file.Service,
	converterService converter.Service,
	scheduler scheduler.Scheduler,
	repoStore store.RepoStore,
	urlProvider url.Provider,
	templateStore store.TemplateStore,
	pluginStore store.PluginStore,
	publicAccess publicaccess.Service,
) Triggerer {
	return New(executionStore, checkStore, stageStore, pipelineStore,
		tx, repoStore, urlProvider, scheduler, fileService, converterService,
		templateStore, pluginStore, publicAccess)
}
