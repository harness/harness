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

func (i infraProvisioner) TriggerStop(
	ctx context.Context,
	infraProviderResource types.InfraProviderResource,
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

	err = infraProvider.Stop(ctx, infra)
	if err != nil {
		return fmt.Errorf("unable to trigger stop infra %+v: %w", infra, err)
	}

	return nil
}

func (i infraProvisioner) ResumeStop(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	stoppedInfra types.Infrastructure,
) error {
	infraProvider, err := i.getInfraProvider(stoppedInfra.ProviderType)
	if err != nil {
		return err
	}

	if infraProvider.ProvisioningType() == enum.InfraProvisioningTypeNew {
		return i.resumeStopForNewProvisioning(ctx, gitspaceConfig, stoppedInfra)
	}

	return nil
}

func (i infraProvisioner) resumeStopForNewProvisioning(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	stoppedInfra types.Infrastructure,
) error {
	err := i.updateInfraProvisionedRecord(ctx, gitspaceConfig, stoppedInfra)
	if err != nil {
		return fmt.Errorf("unable to update provisioned record after stopping: %w", err)
	}

	return nil
}
