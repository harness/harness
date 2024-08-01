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

package gitspace

import (
	"context"
	"fmt"

	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const resourceNotFoundErr = "Failed to find gitspace: resource not found"

func (c *Service) Find(
	ctx context.Context,
	spaceID int64,
	spacePath string,
	identifier string,
) (*types.GitspaceConfig, error) {
	var gitspaceConfigResult *types.GitspaceConfig
	txErr := c.tx.WithTx(ctx, func(ctx context.Context) error {
		gitspaceConfig, err := c.gitspaceConfigStore.FindByIdentifier(ctx, spaceID, identifier)
		if err != nil {
			return fmt.Errorf("failed to find gitspace config: %w", err)
		}
		infraProviderResource, err := c.infraProviderSvc.FindResource(ctx, gitspaceConfig.InfraProviderResourceID)
		if err != nil {
			return fmt.Errorf("failed to find infra provider resource for gitspace config: %w", err)
		}
		gitspaceConfig.SpacePath = spacePath
		gitspaceConfig.InfraProviderResourceIdentifier = infraProviderResource.Identifier
		instance, err := c.gitspaceInstanceStore.FindLatestByGitspaceConfigID(ctx, gitspaceConfig.ID, gitspaceConfig.SpaceID)
		if err != nil && err.Error() != resourceNotFoundErr { // TODO fix this
			return fmt.Errorf("failed to find gitspace instance for config ID : %s %w", gitspaceConfig.Identifier, err)
		}
		if instance != nil {
			gitspaceConfig.GitspaceInstance = instance
			instance.SpacePath = gitspaceConfig.SpacePath
			gitspaceStateType, err := enum.GetGitspaceStateFromInstance(instance.State)
			if err != nil {
				return err
			}
			gitspaceConfig.State = gitspaceStateType
		} else {
			gitspaceConfig.State = enum.GitspaceStateUninitialized
		}
		gitspaceConfigResult = gitspaceConfig
		return nil
	}, dbtx.TxDefaultReadOnly)
	if txErr != nil {
		return nil, txErr
	}
	return gitspaceConfigResult, nil
}
