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
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

func (c *Service) StartGitspaceAction(
	ctx context.Context,
	config types.GitspaceConfig,
) error {
	savedGitspaceInstance, err := c.gitspaceInstanceStore.FindLatestByGitspaceConfigID(ctx, config.ID)
	if err != nil && !errors.Is(err, store.ErrResourceNotFound) {
		return err
	}

	if config.IsMarkedForInfraReset && savedGitspaceInstance != nil && !savedGitspaceInstance.State.IsFinalStatus() {
		savedGitspaceInstance.State = enum.GitspaceInstanceStatePendingCleanup
		err = c.UpdateInstance(ctx, savedGitspaceInstance)
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Msgf(
				"failed to mark old gitspace instance as pending cleanup for config ID: %s",
				config.Identifier,
			)
		}
		// Don't return here - continue to create a new instance
		// The old instance will be cleaned up by the background job
		savedGitspaceInstance = nil // Treat as if no instance exists so a new one is created
	}

	config.GitspaceInstance = savedGitspaceInstance
	err = c.gitspaceBusyOperation(ctx, config)
	if err != nil {
		return err
	}
	if savedGitspaceInstance == nil || savedGitspaceInstance.State.IsFinalStatus() {
		gitspaceInstance, err := c.buildGitspaceInstance(config)
		if err != nil {
			return err
		}

		if savedGitspaceInstance != nil {
			gitspaceInstance.HasGitChanges = savedGitspaceInstance.HasGitChanges
		}

		if err = c.gitspaceInstanceStore.Create(ctx, gitspaceInstance); err != nil {
			return fmt.Errorf("failed to create gitspace instance for %s %w", config.Identifier, err)
		}
	}
	newGitspaceInstance, err := c.gitspaceInstanceStore.FindLatestByGitspaceConfigID(ctx, config.ID)
	if err != nil {
		return fmt.Errorf("failed to find gitspace with config ID : %s %w", config.Identifier, err)
	}
	config.GitspaceInstance = newGitspaceInstance
	c.submitAsyncOps(ctx, config, enum.GitspaceActionTypeStart)
	return nil
}
