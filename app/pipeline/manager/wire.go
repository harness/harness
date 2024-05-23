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

package manager

import (
	"github.com/harness/gitness/app/pipeline/converter"
	"github.com/harness/gitness/app/pipeline/file"
	"github.com/harness/gitness/app/pipeline/scheduler"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/livelog"
	"github.com/harness/gitness/types"

	"github.com/drone/runner-go/client"
	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideExecutionManager,
	ProvideExecutionClient,
)

// ProvideExecutionManager provides an execution manager.
func ProvideExecutionManager(
	config *types.Config,
	executionStore store.ExecutionStore,
	pipelineStore store.PipelineStore,
	urlProvider url.Provider,
	sseStreamer sse.Streamer,
	fileService file.Service,
	converterService converter.Service,
	logStore store.LogStore,
	logStream livelog.LogStream,
	checkStore store.CheckStore,
	repoStore store.RepoStore,
	scheduler scheduler.Scheduler,
	secretStore store.SecretStore,
	stageStore store.StageStore,
	stepStore store.StepStore,
	userStore store.PrincipalStore,
	publicAccess publicaccess.Service,
) ExecutionManager {
	return New(config, executionStore, pipelineStore, urlProvider, sseStreamer, fileService, converterService,
		logStore, logStream, checkStore, repoStore, scheduler, secretStore, stageStore, stepStore, userStore, publicAccess)
}

// ProvideExecutionClient provides a client implementation to interact with the execution manager.
// We use an embedded client here.
func ProvideExecutionClient(
	manager ExecutionManager,
	urlProvider url.Provider,
	config *types.Config,
) client.Client {
	return NewEmbeddedClient(manager, urlProvider, config)
}
