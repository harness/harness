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
	"fmt"

	"github.com/harness/gitness/infraprovider"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func (i infraProvisioner) Find(
	ctx context.Context,
	infraProviderResource types.InfraProviderResource,
	gitspaceConfig types.GitspaceConfig,
	requiredGitspacePorts []int,
) (*types.Infrastructure, error) {
	infraProviderEntity, err := i.getConfigFromResource(ctx, infraProviderResource)
	if err != nil {
		return nil, err
	}

	infraProvider, err := i.getInfraProvider(infraProviderEntity.Type)
	if err != nil {
		return nil, err
	}

	var inputParams []types.InfraProviderParameter
	var agentPort = 0
	if infraProvider.ProvisioningType() == enum.InfraProvisioningTypeNew {
		inputParams, err = i.paramsForProvisioningTypeNew(ctx, gitspaceConfig)
		if err != nil {
			return nil, err
		}

		// TODO: What if the agent port has deviated from when the last instance was created?
		agentPort = i.config.AgentPort
	} else {
		inputParams, err = i.paramsForProvisioningTypeExisting(ctx, infraProviderResource, infraProvider)
		if err != nil {
			return nil, err
		}
	}

	return infraProvider.Find(ctx, gitspaceConfig.SpaceID, gitspaceConfig.SpacePath,
		gitspaceConfig.Identifier, gitspaceConfig.GitspaceInstance.Identifier,
		agentPort, requiredGitspacePorts, inputParams)
}

func (i infraProvisioner) paramsForProvisioningTypeNew(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
) ([]types.InfraProviderParameter, error) {
	infraProvisionedLatest, err := i.infraProvisionedStore.FindLatestByGitspaceInstanceID(
		ctx, gitspaceConfig.SpaceID, gitspaceConfig.GitspaceInstance.ID)
	if err != nil {
		return nil, fmt.Errorf(
			"could not find latest infra provisioned entity for instance %d: %w",
			gitspaceConfig.GitspaceInstance.ID, err)
	}

	allParams := stringToParams(infraProvisionedLatest.InputParams)

	return allParams, nil
}

func (i infraProvisioner) paramsForProvisioningTypeExisting(
	ctx context.Context,
	infraProviderResource types.InfraProviderResource,
	infraProvider infraprovider.InfraProvider,
) ([]types.InfraProviderParameter, error) {
	allParams, err := i.getAllParamsFromDB(ctx, infraProviderResource, infraProvider)
	if err != nil {
		return nil, fmt.Errorf("could not get all params from DB while finding: %w", err)
	}

	return allParams, nil
}
