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

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const gitspaceConfigNotFound = "Failed to find gitspace config with identifier "

func (c *Controller) Delete(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	identifier string,
) error {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return fmt.Errorf("failed to find space: %w", err)
	}
	err = apiauth.CheckGitspace(ctx, c.authorizer, session, space.Path, identifier, enum.PermissionGitspaceDelete)
	if err != nil {
		return fmt.Errorf("failed to authorize: %w", err)
	}
	gitspaceConfig, err := c.gitspaceConfigStore.FindByIdentifier(ctx, space.ID, identifier)
	gitspaceConfig.SpacePath = space.Path
	if err != nil || gitspaceConfig == nil {
		log.Err(err).Msg(gitspaceConfigNotFound + identifier)
		return err
	}
	instance, _ := c.gitspaceInstanceStore.FindLatestByGitspaceConfigID(ctx, gitspaceConfig.ID)
	gitspaceConfig.GitspaceInstance = instance
	if instance == nil || instance.State == enum.GitspaceInstanceStateUninitialized {
		gitspaceConfig.IsDeleted = true
		if err = c.gitspaceSvc.UpdateConfig(ctx, gitspaceConfig); err != nil {
			return fmt.Errorf("failed to mark gitspace config as deleted: %w", err)
		}
	} else {
		ctxWithoutCancel := context.WithoutCancel(ctx)
		go c.removeGitspace(ctxWithoutCancel, *gitspaceConfig)
	}
	return nil
}

func (c *Controller) removeGitspace(ctx context.Context, config types.GitspaceConfig) {
	if config.GitspaceInstance.State == enum.GitspaceInstanceStateRunning {
		activeTimeEnded := time.Now().UnixMilli()
		config.GitspaceInstance.ActiveTimeEnded = &activeTimeEnded
		config.GitspaceInstance.TotalTimeUsed =
			*(config.GitspaceInstance.ActiveTimeEnded) - *(config.GitspaceInstance.ActiveTimeStarted)
		config.GitspaceInstance.State = enum.GitspaceInstanceStateStopping
		err := c.gitspaceSvc.UpdateInstance(ctx, config.GitspaceInstance)
		if err != nil {
			log.Ctx(ctx).Err(err).Msgf("failed to update instance %s before triggering delete",
				config.GitspaceInstance.Identifier)
			return
		}
	}
	if err := c.gitspaceSvc.TriggerDelete(ctx, config); err != nil {
		log.Ctx(ctx).Err(err).Msgf("error during triggering delete for gitspace instance %s",
			config.GitspaceInstance.Identifier)
		config.GitspaceInstance.State = enum.GitspaceInstanceStateError
		if updateErr := c.gitspaceSvc.UpdateInstance(ctx, config.GitspaceInstance); updateErr != nil {
			log.Ctx(ctx).Err(updateErr).Msgf("failed to update instance %s after error in triggering delete",
				config.GitspaceInstance.Identifier)
		}
		return
	}
	log.Ctx(ctx).Debug().Msgf("successfully triggered delete for gitspace instance %s",
		config.GitspaceInstance.Identifier)
}
