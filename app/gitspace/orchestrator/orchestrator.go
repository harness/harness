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
	"github.com/harness/gitness/types/enum"
)

const (
	vscodeURLScheme   = "vscode-remote"
	intellijURLScheme = "jetbrains-gateway"
)

type Orchestrator interface {
	// TriggerStartGitspace fetches the infra resources configured for the gitspace and triggers the infra provisioning.
	TriggerStartGitspace(ctx context.Context, gitspaceConfig types.GitspaceConfig) *types.GitspaceError

	// ResumeStartGitspace saves the provisioned infra, resolves the code repo details & creates the Gitspace container.
	ResumeStartGitspace(
		ctx context.Context,
		gitspaceConfig types.GitspaceConfig,
		provisionedInfra types.Infrastructure,
	) (types.GitspaceInstance, *types.GitspaceError)

	// TriggerStopGitspace stops the Gitspace container and triggers infra deprovisioning to deprovision
	// all the infra resources which are not required to restart the Gitspace.
	TriggerStopGitspace(ctx context.Context, gitspaceConfig types.GitspaceConfig) *types.GitspaceError

	// ResumeStopGitspace saves the deprovisioned infra details.
	ResumeStopGitspace(
		ctx context.Context,
		gitspaceConfig types.GitspaceConfig,
		stoppedInfra types.Infrastructure,
	) (enum.GitspaceInstanceStateType, *types.GitspaceError)

	// TriggerCleanupInstanceResources cleans up all the resources exclusive to gitspace instance.
	TriggerCleanupInstanceResources(ctx context.Context, gitspaceConfig types.GitspaceConfig) error

	// ResumeCleanupInstanceResources saves the cleaned up infra details.
	ResumeCleanupInstanceResources(
		ctx context.Context,
		gitspaceConfig types.GitspaceConfig,
		cleanedUpInfra types.Infrastructure,
	) (enum.GitspaceInstanceStateType, error)

	// TriggerDeleteGitspace removes the Gitspace container and triggers infra deprovisioning to deprovision
	// the infra resources.
	// canDeleteUserData = false -> trigger deprovision of all resources except storage associated to user data.
	// canDeleteUserData = true -> trigger deprovision of all resources.
	TriggerDeleteGitspace(ctx context.Context, gitspaceConfig types.GitspaceConfig, canDeleteUserData bool) error

	// ResumeDeleteGitspace saves the deprovisioned infra details.
	ResumeDeleteGitspace(
		ctx context.Context,
		gitspaceConfig types.GitspaceConfig,
		deprovisionedInfra types.Infrastructure,
	) (enum.GitspaceInstanceStateType, error)

	// GetGitspaceLogs fetches gitspace's start/stop logs.
	GetGitspaceLogs(ctx context.Context, gitspaceConfig types.GitspaceConfig) (string, error)
}
