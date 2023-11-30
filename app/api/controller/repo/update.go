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
	"strings"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

// UpdateInput is used for updating a repo.
type UpdateInput struct {
	Description *string `json:"description"`
	IsPublic    *bool   `json:"is_public"`
}

func (in *UpdateInput) hasChanges(repo *types.Repository) bool {
	return (in.Description != nil && *in.Description != repo.Description) ||
		(in.IsPublic != nil && *in.IsPublic != repo.IsPublic)
}

// Update updates a repository.
func (c *Controller) Update(ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *UpdateInput,
) (*types.Repository, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit, false)
	if err != nil {
		return nil, err
	}

	if !in.hasChanges(repo) {
		return repo, nil
	}

	if err = c.sanitizeUpdateInput(in); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	repo, err = c.repoStore.UpdateOptLock(ctx, repo, func(repo *types.Repository) error {
		// update values only if provided
		if in.Description != nil {
			repo.Description = *in.Description
		}
		if in.IsPublic != nil {
			repo.IsPublic = *in.IsPublic
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// backfill repo url
	repo.GitURL = c.urlProvider.GenerateGITCloneURL(repo.Path)

	return repo, nil
}

func (c *Controller) sanitizeUpdateInput(in *UpdateInput) error {
	if in.IsPublic != nil {
		if *in.IsPublic && !c.publicResourceCreationEnabled {
			return errPublicRepoCreationDisabled
		}
	}

	if in.Description != nil {
		*in.Description = strings.TrimSpace(*in.Description)
		if err := check.Description(*in.Description); err != nil {
			return err
		}
	}

	return nil
}
