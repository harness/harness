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

package infraprovider

import (
	"context"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type InfraProvider interface {
	// Provision provisions infrastructure against a gitspace with the provided parameters.
	Provision(
		ctx context.Context,
		spaceID int64,
		spacePath string,
		gitspaceConfigIdentifier string,
		gitspaceInstanceIdentifier string,
		agentPort int,
		requiredGitspacePorts []int,
		inputParameters []types.InfraProviderParameter,
	) error
	// Find finds infrastructure provisioned against a gitspace.
	Find(
		ctx context.Context,
		spaceID int64,
		spacePath string,
		gitspaceConfigIdentifier string,
		gitspaceInstanceIdentifier string,
		agentPort int,
		requiredGitspacePorts []int,
		inputParameters []types.InfraProviderParameter,
	) (*types.Infrastructure, error)
	// Stop frees up the resources allocated against a gitspace, which can be freed.
	Stop(ctx context.Context, infra types.Infrastructure) error
	// Deprovision removes all infrastructure provisioned against the gitspace.
	Deprovision(ctx context.Context, infra types.Infrastructure) error
	// AvailableParams provides a schema to define the infrastructure.
	AvailableParams() []types.InfraProviderParameterSchema
	// ValidateParams validates the supplied params before defining the infrastructure resource .
	ValidateParams(inputParameters []types.InfraProviderParameter) error
	// TemplateParams provides a list of params which are of type template.
	TemplateParams() []types.InfraProviderParameterSchema
	// ProvisioningType specifies whether the provider will provision new infra resources or it will reuse existing.
	ProvisioningType() enum.InfraProvisioningType
}
