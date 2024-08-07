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

package container

import (
	"context"

	"github.com/harness/gitness/app/gitspace/orchestrator/ide"
	"github.com/harness/gitness/app/gitspace/scm"
	"github.com/harness/gitness/types"
)

type Orchestrator interface {
	// CreateAndStartGitspace starts an exited container and starts a new container if the container is removed.
	// If the container is newly created, it clones the code, sets up the IDE and executes the postCreateCommand.
	// It returns the container ID, name and ports used.
	CreateAndStartGitspace(
		ctx context.Context,
		gitspaceConfig types.GitspaceConfig,
		infra types.Infrastructure,
		resolvedDetails scm.ResolvedDetails,
		defaultBaseImage string,
		ideService ide.IDE,
	) (*StartResponse, error)

	// StopGitspace stops the gitspace container.
	StopGitspace(ctx context.Context, config types.GitspaceConfig, infra types.Infrastructure) error

	// StopAndRemoveGitspace stops and removes the gitspace container.
	StopAndRemoveGitspace(ctx context.Context, config types.GitspaceConfig, infra types.Infrastructure) error

	// Status checks if the infra is reachable and ready to orchestrate containers.
	Status(ctx context.Context, infra types.Infrastructure) error
}
