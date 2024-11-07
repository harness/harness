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
	"time"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func (c *Service) StopGitspaceAction(
	ctx context.Context,
	config *types.GitspaceConfig,
	now time.Time,
) error {
	savedGitspaceInstance, err := c.gitspaceInstanceStore.FindLatestByGitspaceConfigID(ctx, config.ID)
	if err != nil {
		return fmt.Errorf("failed to find gitspace with config ID : %s %w", config.Identifier, err)
	}
	if savedGitspaceInstance.State.IsFinalStatus() {
		return fmt.Errorf("gitspace instance cannot be stopped with ID %s", savedGitspaceInstance.Identifier)
	}
	config.GitspaceInstance = savedGitspaceInstance
	err = c.gitspaceBusyOperation(ctx, config)
	if err != nil {
		return err
	}

	activeTimeEnded := now.UnixMilli()
	config.GitspaceInstance.ActiveTimeEnded = &activeTimeEnded
	config.GitspaceInstance.TotalTimeUsed =
		*(config.GitspaceInstance.ActiveTimeEnded) - *(config.GitspaceInstance.ActiveTimeStarted)
	config.GitspaceInstance.State = enum.GitspaceInstanceStateStopping
	if err = c.UpdateInstance(ctx, config.GitspaceInstance); err != nil {
		return fmt.Errorf("failed to update gitspace config for stopping %s %w", config.Identifier, err)
	}
	c.submitAsyncOps(ctx, config, enum.GitspaceActionTypeStop)
	return nil
}

func (c *Service) GitspaceAutostopAction(
	ctx context.Context,
	config *types.GitspaceConfig,
	now time.Time,
) error {
	c.EmitGitspaceConfigEvent(ctx, config, enum.GitspaceEventTypeGitspaceAutoStop)
	if err := c.StopGitspaceAction(ctx, config, now); err != nil {
		return err
	}
	return nil
}
