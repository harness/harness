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

package orchestrator

import (
	"context"

	"github.com/harness/gitness/types"
)

type Orchestrator interface {
	// StartGitspace is responsible for all the operations necessary to create the Gitspace container. It fetches the
	// devcontainer.json from the code repo, provisions infra using the infra provisioner and setting up the Gitspace
	// through the container orchestrator.
	StartGitspace(ctx context.Context, gitspaceConfig *types.GitspaceConfig) error

	// StopGitspace is responsible for stopping a running Gitspace. It stops the Gitspace container and unprovisions
	// all the infra resources which are not required to restart the Gitspace.
	StopGitspace(ctx context.Context, gitspaceConfig *types.GitspaceConfig) error

	// DeleteGitspace is responsible for deleting a Gitspace. It stops the Gitspace container and unprovisions
	// all the infra resources.
	DeleteGitspace(ctx context.Context, gitspaceConfig *types.GitspaceConfig) (*types.GitspaceInstance, error)
}
