// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// UpdateInput is used for updating a repo.
type UpdateInput struct {
	Description *string `json:"description"`
	IsPublic    *bool   `json:"is_public"`
}

func (in *UpdateInput) hasChanges(repo *types.Repository) bool {
	return (in.Description != nil && repo.Description == *in.Description || in.Description == nil) &&
		(in.IsPublic != nil && repo.IsPublic == *in.IsPublic || in.IsPublic == nil)
}

// Update updates a repository.
func (c *Controller) Update(ctx context.Context, session *auth.Session,
	repoRef string, in *UpdateInput) (*types.Repository, error) {
	repo, err := c.repoStore.FindRepoFromRef(ctx, repoRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoEdit, false); err != nil {
		return nil, err
	}

	// check if anything needs to be changed
	if in.hasChanges(repo) {
		return repo, err
	}

	repo, err = c.repoStore.UpdateOptLock(ctx, repo, func(repo *types.Repository) error {
		// update values only if provided
		if in.Description != nil {
			repo.Description = *in.Description
		}
		if in.IsPublic != nil {
			repo.IsPublic = *in.IsPublic
		}

		// ensure provided values are valid
		if errValidate := c.repoCheck(repo); errValidate != nil {
			return errValidate
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// populate repo url
	repo.GitURL, err = GenerateRepoGitURL(c.gitBaseURL, repo.Path)
	if err != nil {
		return nil, err
	}

	return repo, nil
}
