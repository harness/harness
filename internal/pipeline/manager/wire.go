// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package manager

import (
	"github.com/harness/gitness/internal/pipeline/events"
	"github.com/harness/gitness/internal/pipeline/file"
	"github.com/harness/gitness/internal/pipeline/scheduler"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/url"
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
	urlProvider *url.Provider,
	events events.Events,
	fileService file.FileService,
	logStore store.LogStore,
	logStream livelog.LogStream,
	repoStore store.RepoStore,
	scheduler scheduler.Scheduler,
	secretStore store.SecretStore,
	stageStore store.StageStore,
	stepStore store.StepStore,
	userStore store.PrincipalStore) ExecutionManager {
	return New(config, executionStore, pipelineStore, urlProvider, events, fileService, logStore,
		logStream, repoStore, scheduler, secretStore, stageStore, stepStore, userStore)
}

// ProvideExecutionClient provides a client implementation to interact with the execution manager.
// We use an embedded client here
func ProvideExecutionClient(manager ExecutionManager, config *types.Config) client.Client {
	return NewEmbeddedClient(manager, config)
}
