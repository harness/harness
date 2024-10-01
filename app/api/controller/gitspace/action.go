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
	"strconv"
	"strings"
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

type ActionInput struct {
	Action     enum.GitspaceActionType `json:"action"`
	Identifier string                  `json:"-"`
	SpaceRef   string                  `json:"-"` // Ref of the parent space
}

func (c *Controller) Action(
	ctx context.Context,
	session *auth.Session,
	in *ActionInput,
) (*types.GitspaceConfig, error) {
	if err := c.sanitizeActionInput(in); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}
	space, err := c.spaceStore.FindByRef(ctx, in.SpaceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find space: %w", err)
	}
	err = apiauth.CheckGitspace(ctx, c.authorizer, session, space.Path, in.Identifier, enum.PermissionGitspaceAccess)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize: %w", err)
	}

	gitspaceConfig, err := c.gitspaceConfigStore.FindByIdentifier(ctx, space.ID, in.Identifier)
	gitspaceConfig.SpacePath = space.Path
	gitspaceConfig.SpaceID = space.ID
	if err != nil {
		return nil, fmt.Errorf("failed to find gitspace config: %w", err)
	}

	// check if it's an internal repo
	if gitspaceConfig.CodeRepo.Type == enum.CodeRepoTypeGitness {
		if gitspaceConfig.CodeRepo.Ref == nil {
			return nil, fmt.Errorf("couldn't fetch repo for the user, no ref found: %w", err)
		}
		repo, err := c.repoStore.FindByRef(ctx, *gitspaceConfig.CodeRepo.Ref)
		if err != nil {
			return nil, fmt.Errorf("couldn't fetch repo for the user: %w", err)
		}
		if err = apiauth.CheckRepo(
			ctx,
			c.authorizer,
			session,
			repo,
			enum.PermissionRepoView); err != nil {
			return nil, err
		}
	}
	// All the actions should be idempotent.
	switch in.Action {
	case enum.GitspaceActionTypeStart:
		err = c.gitspaceLimiter.Usage(ctx, space.ID)
		if err != nil {
			return nil, fmt.Errorf("usage has exceeded limit, can not start any gitspaces: %w", err)
		}

		c.gitspaceSvc.EmitGitspaceConfigEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeGitspaceActionStart)
		err = c.gitspaceSvc.StartGitspaceAction(ctx, gitspaceConfig)
		return gitspaceConfig, err
	case enum.GitspaceActionTypeStop:
		c.gitspaceSvc.EmitGitspaceConfigEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeGitspaceActionStop)
		err = c.gitspaceSvc.StopGitspaceAction(ctx, gitspaceConfig, time.Now())
		return gitspaceConfig, err
	default:
		return nil, fmt.Errorf("unknown action %s on gitspace : %s", string(in.Action), gitspaceConfig.Identifier)
	}
}

func (c *Controller) sanitizeActionInput(in *ActionInput) error {
	if err := check.Identifier(in.Identifier); err != nil {
		return err
	}
	parentRefAsID, err := strconv.ParseInt(in.SpaceRef, 10, 64)
	if (err == nil && parentRefAsID <= 0) || (len(strings.TrimSpace(in.SpaceRef)) == 0) {
		return ErrGitspaceRequiresParent
	}
	return nil
}
