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
	"time"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// gitspaceInstanceCleaningTimedOutMins is timeout for which a gitspace instance can be in cleaning state.
const gitspaceInstanceCleaningTimedOutMins = 15

func (c *Service) RemoveGitspace(ctx context.Context, config types.GitspaceConfig, canDeleteUserData bool) {
	if config.GitspaceInstance.State == enum.GitSpaceInstanceStateCleaning &&
		time.Since(time.UnixMilli(config.GitspaceInstance.Updated)).Milliseconds() <=
			(gitspaceInstanceCleaningTimedOutMins*60*1000) {
		log.Ctx(ctx).Warn().Msgf("gitspace start/stop is already pending for : %q",
			config.GitspaceInstance.Identifier)
		return
	}

	if config.GitspaceInstance.State == enum.GitspaceInstanceStateRunning {
		activeTimeEnded := time.Now().UnixMilli()
		config.GitspaceInstance.ActiveTimeEnded = &activeTimeEnded
		config.GitspaceInstance.TotalTimeUsed =
			*(config.GitspaceInstance.ActiveTimeEnded) - *(config.GitspaceInstance.ActiveTimeStarted)
		config.GitspaceInstance.State = enum.GitspaceInstanceStateStopping
	} else {
		config.GitspaceInstance.State = enum.GitSpaceInstanceStateCleaning
	}

	err := c.UpdateInstance(ctx, config.GitspaceInstance)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to update instance %s before triggering delete",
			config.GitspaceInstance.Identifier)
		return
	}

	if err := c.TriggerDelete(ctx, config, canDeleteUserData); err != nil {
		log.Ctx(ctx).Err(err).Msgf("error during triggering delete for gitspace instance %s",
			config.GitspaceInstance.Identifier)
		config.GitspaceInstance.State = enum.GitspaceInstanceStateError
		if updateErr := c.UpdateInstance(ctx, config.GitspaceInstance); updateErr != nil {
			log.Ctx(ctx).Err(updateErr).Msgf("failed to update instance %s after error in triggering delete",
				config.GitspaceInstance.Identifier)
		}
		return
	}
	log.Ctx(ctx).Debug().Msgf("successfully triggered delete for gitspace instance %s",
		config.GitspaceInstance.Identifier)
}

func (c *Service) TriggerDelete(
	ctx context.Context,
	config types.GitspaceConfig,
	canDeleteUserData bool,
) error {
	return c.orchestrator.TriggerDeleteGitspace(ctx, config, canDeleteUserData)
}
