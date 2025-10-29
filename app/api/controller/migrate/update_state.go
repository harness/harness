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

package migrate

import (
	"context"
	"fmt"
	"slices"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

var validTransitions = map[enum.RepoState][]enum.RepoState{
	enum.RepoStateActive:            {enum.RepoStateMigrateDataImport},
	enum.RepoStateMigrateDataImport: {enum.RepoStateActive},
	enum.RepoStateMigrateGitPush:    {enum.RepoStateActive, enum.RepoStateMigrateDataImport},
}

type UpdateStateInput struct {
	State enum.RepoState `json:"state"`
	Force bool           `json:"force,omitempty"`
}

func (c *Controller) UpdateRepoState(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *UpdateStateInput,
) (*types.Repository, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	repoFull, err := c.repoStore.Find(ctx, repo.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo by ID: %w", err)
	}

	repoFull, err = c.repoStore.UpdateOptLock(ctx, repoFull, func(r *types.Repository) error {
		if !stateTransitionValid(ctx, repo.Identifier, r.State, in.State, in.Force) {
			return usererror.BadRequestf("Changing repo state from %s to %s is not allowed.", r.State, in.State)
		}

		r.State = in.State

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update the repo state: %w", err)
	}

	c.repoFinder.MarkChanged(ctx, repo)

	return repoFull, nil
}

func stateTransitionValid(
	ctx context.Context,
	repoIdentifier string,
	currentState enum.RepoState,
	newState enum.RepoState,
	force bool,
) bool {
	if slices.Contains(validTransitions[currentState], newState) {
		return true
	}

	if force {
		log.Ctx(ctx).Warn().Msgf("Forcing state transition for repo %s from %s to %s",
			repoIdentifier, currentState, newState)
		return true
	}

	return false
}
