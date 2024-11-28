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

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/store"
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
	if err != nil || gitspaceConfig == nil {
		log.Err(err).Msg(gitspaceConfigNotFound + identifier)
		return err
	}
	gitspaceConfig.SpacePath = space.Path
	instance, err := c.gitspaceInstanceStore.FindLatestByGitspaceConfigID(ctx, gitspaceConfig.ID)
	if err != nil && !errors.Is(err, store.ErrResourceNotFound) {
		log.Err(err).Msgf("Failed to find latest gitspace instance for config : %s", identifier)
		return err
	}
	gitspaceConfig.GitspaceInstance = instance
	if instance == nil || instance.State == enum.GitspaceInstanceStateUninitialized {
		gitspaceConfig.IsMarkedForDeletion = true
		gitspaceConfig.IsDeleted = true
		if err = c.gitspaceSvc.UpdateConfig(ctx, gitspaceConfig); err != nil {
			return fmt.Errorf("failed to mark gitspace config as deleted: %w", err)
		}

		return nil
	}

	// mark can_delete for gitconfig as true so that if delete operation fails, cron job can clean up resources.
	gitspaceConfig.IsMarkedForDeletion = true
	if err = c.gitspaceSvc.UpdateConfig(ctx, gitspaceConfig); err != nil {
		return fmt.Errorf("failed to mark gitspace config is_marked_for_deletion column: %w", err)
	}

	ctxWithoutCancel := context.WithoutCancel(ctx)
	go func() {
		err2 := c.gitspaceSvc.RemoveGitspace(ctxWithoutCancel, *gitspaceConfig, true)
		if err2 != nil {
			log.Debug().Err(err2).Msgf("unable to Delete gitspace: " + identifier)
		}
	}()
	return nil
}
