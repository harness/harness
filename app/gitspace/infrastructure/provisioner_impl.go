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

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/infraprovider"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

var _ InfraProvisioner = (*infraProvisioner)(nil)

type infraProvisioner struct {
	infraProviderConfigStore   store.InfraProviderConfigStore
	infraProviderResourceStore store.InfraProviderResourceStore
	providerFactory            infraprovider.Factory
}

func NewInfraProvisionerService(
	infraProviderConfigStore store.InfraProviderConfigStore,
	infraProviderResourceStore store.InfraProviderResourceStore,
	providerFactory infraprovider.Factory,
) InfraProvisioner {
	return &infraProvisioner{
		infraProviderConfigStore:   infraProviderConfigStore,
		infraProviderResourceStore: infraProviderResourceStore,
		providerFactory:            providerFactory,
	}
}

func (i infraProvisioner) TriggerProvision(
	ctx context.Context,
	infraProviderResource *types.InfraProviderResource,
	gitspaceConfig types.GitspaceConfig,
	requiredPorts []int,
) error {
	infraProviderEntity, err := i.getConfigFromResource(ctx, infraProviderResource)
	if err != nil {
		return err
	}

	infraProvider, err := i.getInfraProvider(infraProviderEntity.Type)
	if err != nil {
		return err
	}

	if infraProvider.ProvisioningType() == enum.InfraProvisioningTypeNew { //nolint:revive,staticcheck
		// TODO: Check if any existing infra is provisioned, its status and create new infraProvisioned record
	}

	var allParams []types.InfraProviderParameter

	templateParams, err := i.getTemplateParams(infraProvider, infraProviderResource)
	if err != nil {
		return err
	}

	allParams = append(allParams, templateParams...)

	params := i.paramsFromResource(infraProviderResource)

	allParams = append(allParams, params...)

	err = infraProvider.ValidateParams(allParams)
	if err != nil {
		return fmt.Errorf("invalid provisioning params %v: %w", infraProviderResource.Metadata, err)
	}

	err = infraProvider.Provision(
		ctx,
		gitspaceConfig.SpaceID,
		gitspaceConfig.SpacePath,
		gitspaceConfig.Identifier,
		requiredPorts,
		allParams,
	)
	if err != nil {
		return fmt.Errorf(
			"unable to trigger provision infrastructure for gitspaceConfigIdentifier %v: %w",
			gitspaceConfig.Identifier,
			err,
		)
	}

	return nil
}

func (i infraProvisioner) TriggerStop(
	ctx context.Context,
	infraProviderResource *types.InfraProviderResource,
	gitspaceConfig types.GitspaceConfig,
) error {
	infraProviderEntity, err := i.getConfigFromResource(ctx, infraProviderResource)
	if err != nil {
		return err
	}

	infraProvider, err := i.getInfraProvider(infraProviderEntity.Type)
	if err != nil {
		return err
	}

	var allParams []types.InfraProviderParameter

	if infraProvider.ProvisioningType() == enum.InfraProvisioningTypeNew { //nolint:revive
		// TODO: Fetch and check existing infraProvisioned record
		// TODO: If infra was newly provisioned, all params should be available in the provisioned infra record
	} else {
		templateParams, err2 := i.getTemplateParams(infraProvider, infraProviderResource)
		if err2 != nil {
			return err2
		}
		allParams = append(allParams, templateParams...)

		params := i.paramsFromResource(infraProviderResource)

		allParams = append(allParams, params...)
	}

	err = infraProvider.ValidateParams(allParams)
	if err != nil {
		return fmt.Errorf("invalid provisioning params %+v: %w", infraProviderResource.Metadata, err)
	}

	var provisionedInfra *types.Infrastructure
	if infraProvider.ProvisioningType() == enum.InfraProvisioningTypeNew { //nolint:revive
		// TODO: Fetch and check existing infraProvisioned record
	} else {
		provisionedInfra = &types.Infrastructure{
			SpaceID:      gitspaceConfig.SpaceID,
			SpacePath:    gitspaceConfig.SpacePath,
			ResourceKey:  gitspaceConfig.Identifier,
			ProviderType: infraProviderEntity.Type,
			Parameters:   allParams,
		}
	}

	err = infraProvider.Stop(ctx, provisionedInfra)
	if err != nil {
		return fmt.Errorf("unable to trigger stop infra %+v: %w", provisionedInfra, err)
	}

	return nil
}

func (i infraProvisioner) TriggerDeprovision(
	ctx context.Context,
	infraProviderResource *types.InfraProviderResource,
	gitspaceConfig types.GitspaceConfig,
) error {
	infraProviderEntity, err := i.getConfigFromResource(ctx, infraProviderResource)
	if err != nil {
		return err
	}

	infraProvider, err := i.getInfraProvider(infraProviderEntity.Type)
	if err != nil {
		return err
	}

	var allParams []types.InfraProviderParameter

	if infraProvider.ProvisioningType() == enum.InfraProvisioningTypeNew { //nolint:revive
		// TODO: Fetch and check existing infraProvisioned record
		// TODO: If infra was newly provisioned, all params should be available in the provisioned infra record
	} else {
		templateParams, err2 := i.getTemplateParams(infraProvider, infraProviderResource)
		if err2 != nil {
			return err2
		}
		allParams = append(allParams, templateParams...)

		params := i.paramsFromResource(infraProviderResource)

		allParams = append(allParams, params...)
	}

	err = infraProvider.ValidateParams(allParams)
	if err != nil {
		return fmt.Errorf("invalid provisioning params %+v: %w", infraProviderResource.Metadata, err)
	}

	var provisionedInfra *types.Infrastructure
	if infraProvider.ProvisioningType() == enum.InfraProvisioningTypeNew { //nolint:revive
		// TODO: Fetch and check existing infraProvisioned record
	} else {
		provisionedInfra, err = infraProvider.Find(
			ctx, gitspaceConfig.SpaceID, gitspaceConfig.SpacePath, gitspaceConfig.Identifier, allParams)
		if err != nil {
			return fmt.Errorf("unable to find provisioned infra for gitspace %s: %w",
				gitspaceConfig.Identifier, err)
		}
	}
	err = infraProvider.Deprovision(ctx, provisionedInfra)
	if err != nil {
		return fmt.Errorf("unable to trigger deprovision infra %+v: %w", provisionedInfra, err)
	}

	return err
}

func (i infraProvisioner) Find(
	ctx context.Context,
	infraProviderResource *types.InfraProviderResource,
	_ types.GitspaceConfig,
) (*types.Infrastructure, error) {
	infraProviderEntity, err := i.getConfigFromResource(ctx, infraProviderResource)
	if err != nil {
		return nil, err
	}

	infraProvider, err := i.getInfraProvider(infraProviderEntity.Type)
	if err != nil {
		return nil, err
	}

	var infra types.Infrastructure
	if infraProvider.ProvisioningType() == enum.InfraProvisioningTypeNew { //nolint:revive
		// TODO: Fetch existing infraProvisioned record and map to &infraprovider.Infrastructure
	} else {
		var allParams []types.InfraProviderParameter

		templateParams, err2 := i.getTemplateParams(infraProvider, infraProviderResource)
		if err2 != nil {
			return nil, err2
		}
		allParams = append(allParams, templateParams...)

		params := i.paramsFromResource(infraProviderResource)

		allParams = append(allParams, params...)

		infra = types.Infrastructure{
			ProviderType: infraProviderEntity.Type,
			Parameters:   allParams,
			Status:       enum.InfraStatusProvisioned,
		}
	}

	return &infra, nil
}

func (i infraProvisioner) getConfigFromResource(
	ctx context.Context,
	infraProviderResource *types.InfraProviderResource,
) (*types.InfraProviderConfig, error) {
	config, err := i.infraProviderConfigStore.Find(ctx, infraProviderResource.InfraProviderConfigID)
	if err != nil {
		return nil, fmt.Errorf(
			"unable to get infra provider details for ID %d: %w", infraProviderResource.InfraProviderConfigID, err)
	}
	return config, nil
}

func (i infraProvisioner) getInfraProvider(
	infraProviderType enum.InfraProviderType,
) (infraprovider.InfraProvider, error) {
	infraProvider, err := i.providerFactory.GetInfraProvider(infraProviderType)
	if err != nil {
		return nil, fmt.Errorf("unable to get infra provider of type %v: %w", infraProviderType, err)
	}
	return infraProvider, nil
}

func (i infraProvisioner) getTemplateParams(
	infraProvider infraprovider.InfraProvider,
	_ *types.InfraProviderResource,
) ([]types.InfraProviderParameter, error) { //nolint:unparam
	templateParams := infraProvider.TemplateParams()
	if len(templateParams) > 0 { //nolint:revive,staticcheck
		// TODO: Fetch templates and convert into []Parameters
	}
	return nil, nil
}

func (i infraProvisioner) paramsFromResource(
	infraProviderResource *types.InfraProviderResource,
) []types.InfraProviderParameter {
	params := make([]types.InfraProviderParameter, len(infraProviderResource.Metadata))
	counter := 0
	for key, value := range infraProviderResource.Metadata {
		if key == "" || value == "" {
			continue
		}
		params[counter] = types.InfraProviderParameter{
			Name:  key,
			Value: value,
		}
		counter++
	}
	return params
}

func (i infraProvisioner) ResumeProvision(
	_ context.Context,
	_ *types.InfraProviderResource,
	_ types.GitspaceConfig,
	_ []int,
	provisionedInfra *types.Infrastructure,
) (*types.Infrastructure, error) {
	infraProvider, err := i.getInfraProvider(provisionedInfra.ProviderType)
	if err != nil {
		return nil, err
	}

	if infraProvider.ProvisioningType() == enum.InfraProvisioningTypeNew { //nolint:revive,staticcheck
		// TODO: Update the infraProvisioned record
	}

	return provisionedInfra, nil
}

func (i infraProvisioner) ResumeStop(
	_ context.Context,
	_ *types.InfraProviderResource,
	_ types.GitspaceConfig,
	stoppedInfra *types.Infrastructure,
) (*types.Infrastructure, error) {
	infraProvider, err := i.getInfraProvider(stoppedInfra.ProviderType)
	if err != nil {
		return nil, err
	}

	if infraProvider.ProvisioningType() == enum.InfraProvisioningTypeNew { //nolint:revive,staticcheck
		// TODO: Update existing infraProvisioned record
	}

	return stoppedInfra, err
}

func (i infraProvisioner) ResumeDeprovision(
	_ context.Context,
	_ *types.InfraProviderResource,
	_ types.GitspaceConfig,
	deprovisionedInfra *types.Infrastructure,
) (*types.Infrastructure, error) {
	infraProvider, err := i.getInfraProvider(deprovisionedInfra.ProviderType)
	if err != nil {
		return nil, err
	}

	if infraProvider.ProvisioningType() == enum.InfraProvisioningTypeNew { //nolint:revive,staticcheck
		// TODO: Update existing infraProvisioned record
	}

	return deprovisionedInfra, err
}
