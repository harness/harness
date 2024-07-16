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

	"github.com/harness/gitness/infraprovider"
	"github.com/harness/gitness/types"
)

// TODO Check if the interface can be discarded

type InfraProvisioner interface {
	// Provision provisions infra resources using the infraProviderResource with different infra providers and
	// stores the details in the db depending on the provisioning type.
	Provision(
		ctx context.Context,
		infraProviderResource *types.InfraProviderResource,
		gitspaceConfig *types.GitspaceConfig,
	) (*infraprovider.Infrastructure, error)

	// Stop unprovisions those resources which can be stopped without losing the gitspace data.
	Stop(
		ctx context.Context,
		infraProviderResource *types.InfraProviderResource,
		gitspaceConfig *types.GitspaceConfig,
	) (*infraprovider.Infrastructure, error)

	// Deprovision removes all the resources created for the gitspace.
	Deprovision(
		ctx context.Context,
		infraProviderResource *types.InfraProviderResource,
		gitspaceConfig *types.GitspaceConfig,
	) (*infraprovider.Infrastructure, error)

	// Find finds the provisioned infra resources for the gitspace instance.
	Find(
		ctx context.Context,
		infraProviderResource *types.InfraProviderResource,
		gitspaceConfig *types.GitspaceConfig,
	) (*infraprovider.Infrastructure, error)
}
