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

package infrastructure

import (
	"context"

	"github.com/harness/gitness/types"
)

// TODO Check if the interface can be discarded

type InfraProvisioner interface {
	// TriggerProvision triggers the provisionining of infra resources using the infraProviderResource with different
	// infra providers.
	TriggerProvision(
		ctx context.Context,
		infraProviderResource types.InfraProviderResource,
		gitspaceConfig types.GitspaceConfig,
		requiredGitspacePorts []int,
	) error

	// ResumeProvision stores the provisioned infra details in the db depending on the provisioning type.
	ResumeProvision(
		ctx context.Context,
		gitspaceConfig types.GitspaceConfig,
		provisionedInfra types.Infrastructure,
	) error

	// TriggerStop triggers deprovisioning of those resources which can be stopped without losing the Gitspace data.
	TriggerStop(
		ctx context.Context,
		infraProviderResource types.InfraProviderResource,
		infra types.Infrastructure,
	) error

	// ResumeStop stores the deprovisioned infra details in the db depending on the provisioning type.
	ResumeStop(
		ctx context.Context,
		gitspaceConfig types.GitspaceConfig,
		deprovisionedInfra types.Infrastructure,
	) error

	// TriggerDeprovision triggers deprovisionign of all the resources created for the Gitspace.
	TriggerDeprovision(
		ctx context.Context,
		infraProviderResource types.InfraProviderResource,
		gitspaceConfig types.GitspaceConfig,
		infra types.Infrastructure,
	) error

	// ResumeDeprovision stores the deprovisioned infra details in the db depending on the provisioning type.
	ResumeDeprovision(
		ctx context.Context,
		gitspaceConfig types.GitspaceConfig,
		deprovisionedInfra types.Infrastructure,
	) error

	// Find finds the provisioned infra resources for the gitspace instance.
	Find(
		ctx context.Context,
		infraProviderResource types.InfraProviderResource,
		gitspaceConfig types.GitspaceConfig,
		requiredGitspacePorts []int,
	) (*types.Infrastructure, error)
}
