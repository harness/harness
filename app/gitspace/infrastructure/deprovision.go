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

func (i infraProvisioner) TriggerDeprovision(
	ctx context.Context,
	infraProviderResource types.InfraProviderResource,
	gitspaceConfig types.GitspaceConfig,
	infra types.Infrastructure,
) error {
	infraProviderEntity, err := i.getConfigFromResource(ctx, infraProviderResource)
	if err != nil {
		return err
	}

	infraProvider, err := i.getInfraProvider(infraProviderEntity.Type)
	if err != nil {
		return err
	}

	if infraProvider.ProvisioningType() == enum.InfraProvisioningTypeNew {
		return i.triggerDeprovisionForNewProvisioning(ctx, infraProvider, gitspaceConfig, infra)
	}
	return i.triggerDeprovisionForExistingProvisioning(ctx, infraProvider, infra)
}

func (i infraProvisioner) triggerDeprovisionForNewProvisioning(
	ctx context.Context,
	infraProvider infraprovider.InfraProvider,
	gitspaceConfig types.GitspaceConfig,
	infra types.Infrastructure,
) error {
	infraProvisionedLatest, err := i.infraProvisionedStore.FindLatestByGitspaceInstanceID(
		ctx, gitspaceConfig.SpaceID, gitspaceConfig.GitspaceInstance.ID)
	if err != nil {
		return fmt.Errorf(
			"could not find latest infra provisioned entity for instance %d: %w",
			gitspaceConfig.GitspaceInstance.ID, err)
	}

	if infraProvisionedLatest.InfraStatus != enum.InfraStatusProvisioned &&
		infraProvisionedLatest.InfraStatus != enum.InfraStatusUnknown {
		return fmt.Errorf("the infrastructure with identifier %s doesn't exist", infra.Identifier)
	}

	err = infraProvider.Deprovision(ctx, infra)
	if err != nil {
		return fmt.Errorf("unable to trigger deprovision infra %+v: %w", infra, err)
	}

	return err
}

func (i infraProvisioner) triggerDeprovisionForExistingProvisioning(
	ctx context.Context,
	infraProvider infraprovider.InfraProvider,
	infra types.Infrastructure,
) error {
	err := infraProvider.Deprovision(ctx, infra)
	if err != nil {
		return fmt.Errorf("unable to trigger deprovision infra %+v: %w", infra, err)
	}

	return err
}

func (i infraProvisioner) ResumeDeprovision(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	deprovisionedInfra types.Infrastructure,
) error {
	infraProvider, err := i.getInfraProvider(deprovisionedInfra.ProviderType)
	if err != nil {
		return err
	}

	if infraProvider.ProvisioningType() == enum.InfraProvisioningTypeNew {
		return i.resumeDeprovisionForNewProvisioning(ctx, gitspaceConfig, deprovisionedInfra)
	}
	return nil
}

func (i infraProvisioner) resumeDeprovisionForNewProvisioning(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	deprovisionedInfra types.Infrastructure,
) error {
	err := i.updateInfraProvisionedRecord(ctx, gitspaceConfig, deprovisionedInfra)
	if err != nil {
		return fmt.Errorf("unable to update provisioned record after deprovisioning: %w", err)
	}

	return nil
}
