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

package metric

import (
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideCollector,
)

func ProvideCollector(
	config *types.Config,
	userStore store.PrincipalStore,
	repoStore store.RepoStore,
	pipelineStore store.PipelineStore,
	executionStore store.ExecutionStore,
	scheduler *job.Scheduler,
	executor *job.Executor,
	gitspaceConfigStore store.GitspaceConfigStore,
) (*Collector, error) {
	job := &Collector{
		hostname:            config.InstanceID,
		enabled:             config.Metric.Enabled,
		endpoint:            config.Metric.Endpoint,
		token:               config.Metric.Token,
		userStore:           userStore,
		repoStore:           repoStore,
		pipelineStore:       pipelineStore,
		executionStore:      executionStore,
		scheduler:           scheduler,
		gitspaceConfigStore: gitspaceConfigStore,
	}

	err := executor.Register(jobType, job)
	if err != nil {
		return nil, err
	}

	return job, nil
}
