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

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func (i infraProvisioner) TriggerCleanupInstance(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	infra types.Infrastructure,
) error {
	infraProviderEntity, err := i.getConfigFromResource(ctx, gitspaceConfig.InfraProviderResource)
	if err != nil {
		return err
	}

	infraProvider, err := i.getInfraProvider(infraProviderEntity.Type)
	if err != nil {
		return err
	}

	return infraProvider.CleanupInstanceResources(ctx, infra)
}

func (i infraProvisioner) ResumeCleanupInstance(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	cleanedInfra types.Infrastructure,
) error {
	infraProvider, err := i.getInfraProvider(cleanedInfra.ProviderType)
	if err != nil {
		return err
	}

	if infraProvider.ProvisioningType() == enum.InfraProvisioningTypeNew {
		return i.resumeCleanupForNewProvisioning(ctx, gitspaceConfig, cleanedInfra)
	}
	return nil
}

func (i infraProvisioner) resumeCleanupForNewProvisioning(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	cleanedInfra types.Infrastructure,
) error {
	err := i.updateInfraProvisionedRecord(ctx, gitspaceConfig, cleanedInfra)
	if err != nil {
		return fmt.Errorf("unable to update provisioned record after cleanup: %w", err)
	}

	return nil
}
