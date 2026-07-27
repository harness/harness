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

package gitspaceoperationsevent

import (
	"context"

	"github.com/harness/gitness/app/gitspace/orchestrator/container/response"
	"github.com/harness/gitness/types"
)

// Orchestrator is the subset of the gitspace orchestrator used by this service.
// Depending on this narrow interface (instead of the concrete, docker/oras-heavy
// orchestrator) keeps that package out of this service's import graph. The
// concrete orchestrator.Orchestrator satisfies it structurally.
type Orchestrator interface {
	FinishResumeStartGitspace(
		ctx context.Context,
		gitspaceConfig types.GitspaceConfig,
		provisionedInfra types.Infrastructure,
		startResponse *response.StartResponse,
	) (types.GitspaceInstance, *types.GitspaceError)
	FinishStopGitspaceContainer(
		ctx context.Context,
		gitspaceConfig types.GitspaceConfig,
		infra types.Infrastructure,
		stopResponse *response.StopResponse,
	) *types.GitspaceError
	FinishRemoveGitspaceContainer(
		ctx context.Context,
		gitspaceConfig types.GitspaceConfig,
		infra types.Infrastructure,
		deleteResponse *response.DeleteResponse,
	) *types.GitspaceError
}
