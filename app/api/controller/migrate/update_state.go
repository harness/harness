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

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

var validTransitions = map[enum.RepoState][]enum.RepoState{
	enum.RepoStateActive:            {enum.RepoStateMigrateDataImport},
	enum.RepoStateMigrateDataImport: {enum.RepoStateActive},
	enum.RepoStateMigrateGitPush:    {enum.RepoStateActive, enum.RepoStateMigrateDataImport},
}

type UpdateStateInput struct {
	State enum.RepoState `json:"state"`
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

	if !stateTransitionValid(repo, in.State) {
		return nil, usererror.BadRequestf("Changing repo state from %s to %s is not allowed.", repo.State, in.State)
	}

	repo, err = c.repoStore.UpdateOptLock(ctx, repo, func(r *types.Repository) error {
		r.State = in.State
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update the repo state: %w", err)
	}

	return repo, nil
}

func stateTransitionValid(repo *types.Repository, newState enum.RepoState) bool {
	for _, validState := range validTransitions[repo.State] {
		if validState == newState {
			return true
		}
	}

	return false
}
