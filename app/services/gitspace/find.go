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
	"errors"
	"fmt"

	"github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func (c *Service) Find(
	ctx context.Context,
	spaceRef string,
	identifier string,
) (*types.GitspaceConfig, error) {
	var gitspaceConfigResult *types.GitspaceConfig
	txErr := c.tx.WithTx(ctx, func(ctx context.Context) error {
		space, err := c.spaceStore.FindByRef(ctx, spaceRef)
		if err != nil {
			return fmt.Errorf("failed to find space: %w", err)
		}
		gitspaceConfig, err := c.gitspaceConfigStore.FindByIdentifier(ctx, space.ID, identifier)
		if err != nil {
			return fmt.Errorf("failed to find gitspace config: %w", err)
		}
		instance, err := c.gitspaceInstanceStore.FindLatestByGitspaceConfigID(ctx, gitspaceConfig.ID)
		if err != nil && !errors.Is(err, store.ErrResourceNotFound) {
			return err
		}
		if instance != nil {
			gitspaceConfig.GitspaceInstance = instance
			instance.SpacePath = gitspaceConfig.SpacePath
			gitspaceStateType, err := enum.GetGitspaceStateFromInstance(instance.State, instance.Updated)
			if err != nil {
				return err
			}
			gitspaceConfig.State = gitspaceStateType
		} else {
			gitspaceConfig.State = enum.GitspaceStateUninitialized
		}
		gitspaceConfigResult = gitspaceConfig
		gitspaceConfig.SpacePath = space.Path
		return nil
	}, dbtx.TxDefaultReadOnly)
	if txErr != nil {
		return nil, txErr
	}
	return gitspaceConfigResult, nil
}
