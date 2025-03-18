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

	events "github.com/harness/gitness/app/events/gitspacedelete"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

func (c *Service) DeleteGitspaceByIdentifier(ctx context.Context, spaceRef string, identifier string) error {
	gitspaceConfig, err := c.FindWithLatestInstanceWithSpacePath(ctx, spaceRef, identifier)
	if err != nil {
		log.Err(err).Msgf("Failed to find latest gitspace config : %s", identifier)
		return err
	}
	return c.deleteGitspace(ctx, gitspaceConfig)
}

func (c *Service) deleteGitspace(ctx context.Context, gitspaceConfig *types.GitspaceConfig) error {
	if gitspaceConfig.GitspaceInstance == nil ||
		gitspaceConfig.GitspaceInstance.State == enum.GitspaceInstanceStateUninitialized {
		gitspaceConfig.IsMarkedForDeletion = true
		gitspaceConfig.IsDeleted = true
		if err := c.UpdateConfig(ctx, gitspaceConfig); err != nil {
			return fmt.Errorf("failed to mark gitspace config as deleted: %w", err)
		}

		return nil
	}

	// mark can_delete for gitconfig as true so that if delete operation fails, cron job can clean up resources.
	gitspaceConfig.IsMarkedForDeletion = true
	if err := c.UpdateConfig(ctx, gitspaceConfig); err != nil {
		return fmt.Errorf("failed to mark gitspace config is_marked_for_deletion column: %w", err)
	}

	c.gitspaceDeleteEventReporter.EmitGitspaceDeleteEvent(ctx, events.GitspaceDeleteEvent,
		&events.GitspaceDeleteEventPayload{GitspaceConfigIdentifier: gitspaceConfig.Identifier,
			SpaceID: gitspaceConfig.SpaceID})

	return nil
}

func (c *Service) RemoveGitspace(ctx context.Context, config types.GitspaceConfig, canDeleteUserData bool) error {
	if config.GitspaceInstance.State == enum.GitSpaceInstanceStateCleaning &&
		time.Since(time.UnixMilli(config.GitspaceInstance.Updated)).Milliseconds() <=
			(gitspaceInstanceCleaningTimedOutMins*60*1000) {
		log.Ctx(ctx).Warn().Msgf("gitspace cleaning is already pending for : %q",
			config.GitspaceInstance.Identifier)
		return fmt.Errorf("gitspace is already pending for : %q", config.GitspaceInstance.Identifier)
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
		return fmt.Errorf("failed to update instance %s before triggering delete: %w",
			config.GitspaceInstance.Identifier,
			err,
		)
	}

	if err := c.orchestrator.TriggerDeleteGitspace(ctx, config, canDeleteUserData); err != nil {
		log.Ctx(ctx).Err(err).Msgf("error during triggering delete for gitspace instance %s",
			config.GitspaceInstance.Identifier)
		config.GitspaceInstance.State = enum.GitspaceInstanceStateError
		if updateErr := c.UpdateInstance(ctx, config.GitspaceInstance); updateErr != nil {
			log.Ctx(ctx).Err(updateErr).Msgf("failed to update instance %s after error in triggering delete",
				config.GitspaceInstance.Identifier)
		}

		return fmt.Errorf("failed to trigger delete for gitspace instance %s: %w",
			config.GitspaceInstance.Identifier,
			err,
		)
	}

	log.Ctx(ctx).Debug().Msgf("successfully triggered delete for gitspace instance %s",
		config.GitspaceInstance.Identifier)

	return nil
}
