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

package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/app/bootstrap"
	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/app/githook"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"

	"github.com/rs/zerolog/log"
)

// handleUpdateDefaultBranch handles git update default branch using branch name from db (not event payload).
func (s *Service) handleUpdateDefaultBranch(
	ctx context.Context,
	event *events.Event[*repoevents.DefaultBranchUpdatedPayload],
) error {
	// the max time we give an update default branch to succeed
	const timeout = 2 * time.Minute
	unlock, err := s.locker.LockDefaultBranch(
		ctx,
		event.Payload.RepoID,
		event.Payload.NewName,  // only used for logging
		timeout+30*time.Second, // add 30s to the lock to give enough time for updating default branch
	)
	if err != nil {
		return fmt.Errorf("failed to lock repo for updating default branch to %s", event.Payload.NewName)
	}
	defer unlock()

	repo, err := s.repoStore.Find(ctx, event.Payload.RepoID)
	if err != nil {
		return fmt.Errorf("update default branch handler failed to find the repo: %w", err)
	}

	// create new, time-restricted context to guarantee update completion, even if request is canceled.
	// TODO: a proper error handling solution required.
	ctx, cancel := context.WithTimeout(
		ctx,
		timeout,
	)
	defer cancel()

	systemPrincipal := bootstrap.NewSystemServiceSession().Principal
	envVars, err := githook.GenerateEnvironmentVariables(
		ctx,
		s.urlProvider.GetInternalAPIURL(ctx),
		repo.ID,
		systemPrincipal.ID,
		true,
		true,
	)
	if err != nil {
		return fmt.Errorf("failed to generate git hook env variables: %w", err)
	}

	err = s.git.UpdateDefaultBranch(ctx, &git.UpdateDefaultBranchParams{
		WriteParams: git.WriteParams{
			Actor: git.Identity{
				Name:  systemPrincipal.DisplayName,
				Email: systemPrincipal.Email,
			},
			RepoUID: repo.GitUID,
			EnvVars: envVars,
		},
		BranchName: repo.DefaultBranch,
	})
	if err != nil {
		return fmt.Errorf("failed to update the repo default branch to %s", repo.DefaultBranch)
	}

	log.Ctx(ctx).Info().Msgf("git repo default branch updated to %s by default branch event handler", repo.DefaultBranch)

	return nil
}
