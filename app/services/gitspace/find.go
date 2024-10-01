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
		gitspaceConfigResult = gitspaceConfig
		if err = c.setInstance(ctx, gitspaceConfigResult, space); err != nil {
			return err
		}
		return nil
	}, dbtx.TxDefaultReadOnly)
	if txErr != nil {
		return nil, txErr
	}
	return gitspaceConfigResult, nil
}

func (c *Service) setInstance(
	ctx context.Context,
	gitspaceConfig *types.GitspaceConfig,
	space *types.Space,
) error {
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
	gitspaceConfig.SpacePath = space.Path
	return nil
}

func (c *Service) FindByID(
	ctx context.Context,
	id int64,
) (*types.GitspaceConfig, error) {
	var gitspaceConfigResult *types.GitspaceConfig
	txErr := c.tx.WithTx(ctx, func(ctx context.Context) error {
		gitspaceConfig, err := c.gitspaceConfigStore.Find(ctx, id)
		gitspaceConfigResult = gitspaceConfig
		if err != nil {
			return fmt.Errorf("failed to find gitspace config: %w", err)
		}
		space, err := c.spaceStore.Find(ctx, gitspaceConfigResult.SpaceID)
		if err != nil {
			return fmt.Errorf("failed to find space: %w", err)
		}
		return c.setInstance(ctx, gitspaceConfigResult, space)
	}, dbtx.TxDefaultReadOnly)
	if txErr != nil {
		return nil, txErr
	}
	return gitspaceConfigResult, nil
}
