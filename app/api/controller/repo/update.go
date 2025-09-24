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
	"encoding/json"
	"fmt"
	"strings"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
	"golang.org/x/exp/slices"
)

// UpdateInput is used for updating a repo.
type UpdateInput struct {
	Description *string         `json:"description"`
	State       *enum.RepoState `json:"state"`
	Tags        *[]string       `json:"tags"`
}

var allowedRepoStateTransitions = map[enum.RepoState][]enum.RepoState{
	enum.RepoStateActive:            {enum.RepoStateArchived, enum.RepoStateMigrateDataImport},
	enum.RepoStateArchived:          {enum.RepoStateActive},
	enum.RepoStateMigrateDataImport: {enum.RepoStateActive},
	enum.RepoStateMigrateGitPush:    {enum.RepoStateActive, enum.RepoStateMigrateDataImport},
}

// Update updates a repository.
func (c *Controller) Update(ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *UpdateInput,
) (*RepositoryOutput, error) {
	repoCore, err := GetRepo(ctx, c.repoFinder, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo: %w", err)
	}

	var additionalAllowedRepoStates []enum.RepoState

	if in.State != nil {
		additionalAllowedRepoStates = []enum.RepoState{
			enum.RepoStateArchived, enum.RepoStateMigrateDataImport, enum.RepoStateMigrateGitPush}
	}

	err = apiauth.CheckRepoState(ctx, session, repoCore, enum.PermissionRepoEdit, additionalAllowedRepoStates...)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repoCore, enum.PermissionRepoEdit); err != nil {
		return nil, fmt.Errorf("access check failed: %w", err)
	}

	repo, err := c.repoStore.Find(ctx, repoCore.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to find repository by ID: %w", err)
	}

	if err = c.sanitizeUpdateInput(in); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	if !in.hasChanges(repo) {
		return GetRepoOutput(ctx, c.publicAccess, repo)
	}

	if err = c.repoCheck.LifecycleRestriction(ctx, session, repoCore); err != nil {
		return nil, err
	}

	if in.State != nil &&
		!slices.Contains(allowedRepoStateTransitions[repo.State], *in.State) {
		return nil, usererror.BadRequestf("Changing the state of a repository from %s to %s is not allowed.",
			repo.State, *in.State)
	}

	var repoClone types.Repository

	repo, err = c.repoStore.UpdateOptLock(ctx, repo, func(repo *types.Repository) error {
		repoClone = *repo

		// update values only if provided
		if in.Description != nil {
			repo.Description = *in.Description
		}
		if in.State != nil {
			repo.State = *in.State
		}
		if in.Tags != nil {
			tags, _ := json.Marshal(in.Tags)
			repo.Tags = tags
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update the repo: %w", err)
	}

	c.repoFinder.MarkChanged(ctx, repo.Core())

	err = c.auditService.Log(ctx,
		session.Principal,
		audit.NewResource(audit.ResourceTypeRepositorySettings, repo.Identifier),
		audit.ActionUpdated,
		paths.Parent(repo.Path),
		audit.WithOldObject(repoClone),
		audit.WithNewObject(repo),
	)
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("failed to insert audit log for update repository operation: %s", err)
	}

	// backfill repo url
	repo.GitURL = c.urlProvider.GenerateGITCloneURL(ctx, repo.Path)
	repo.GitSSHURL = c.urlProvider.GenerateGITCloneSSHURL(ctx, repo.Path)

	if repo.State != repoClone.State {
		c.eventReporter.StateChanged(ctx, &repoevents.StateChangedPayload{
			Base:     eventBase(repo.Core(), &session.Principal),
			OldState: repoClone.State,
			NewState: repo.State,
		})
	}

	return GetRepoOutput(ctx, c.publicAccess, repo)
}

func (in *UpdateInput) hasChanges(repo *types.Repository) bool {
	if in.Description != nil && *in.Description != repo.Description {
		return true
	}
	if in.State != nil && *in.State != repo.State {
		return true
	}
	if hasTagChanges(in, repo) {
		return true
	}

	return false
}

func hasTagChanges(in *UpdateInput, repo *types.Repository) bool {
	if in.Tags == nil {
		return false
	}

	var repoTags []string
	_ = json.Unmarshal(repo.Tags, &repoTags)
	if len(*in.Tags) != len(repoTags) {
		return true
	}

	tagSet := make(map[string]struct{}, len(repoTags))
	for _, t := range repoTags {
		tagSet[t] = struct{}{}
	}
	for _, t := range *in.Tags {
		if _, ok := tagSet[t]; !ok {
			return true
		}
	}

	return false
}

func (c *Controller) sanitizeUpdateInput(in *UpdateInput) error {
	if in.Description != nil {
		*in.Description = strings.TrimSpace(*in.Description)
		if err := check.Description(*in.Description); err != nil {
			return err
		}
	}

	err := sanitizeTags(in.Tags)
	if err != nil {
		return fmt.Errorf("failed to sanitize tags: %w", err)
	}

	return nil
}
