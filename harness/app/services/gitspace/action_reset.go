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

func (c *Service) ResetGitspaceAction(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
) error {
	if gitspaceConfig.GitspaceInstance.State == enum.GitspaceInstanceStateRunning {
		activeTimeEnded := time.Now().UnixMilli()
		gitspaceConfig.GitspaceInstance.ActiveTimeEnded = &activeTimeEnded
		gitspaceConfig.GitspaceInstance.TotalTimeUsed =
			*(gitspaceConfig.GitspaceInstance.ActiveTimeEnded) - *(gitspaceConfig.GitspaceInstance.ActiveTimeStarted)
	}
	gitspaceConfig.IsMarkedForReset = true
	if err := c.UpdateConfig(ctx, &gitspaceConfig); err != nil {
		return fmt.Errorf("failed to update gitspace config for resetting: %w", err)
	}
	c.submitAsyncOps(ctx, gitspaceConfig, enum.GitspaceActionTypeReset)
	return nil
}
