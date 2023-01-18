// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"fmt"
	"strings"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
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
func (c *Controller) Update(ctx context.Context, session *auth.Session,
	repoRef string, in *UpdateInput) (*types.Repository, error) {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoEdit, false); err != nil {
		return nil, err
	}

	if !in.hasChanges(repo) {
		return repo, nil
	}

	if err = sanitizeUpdateInput(in); err != nil {
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
	repo.GitURL = c.urlProvider.GenerateRepoCloneURL(repo.Path)

	return repo, nil
}

func sanitizeUpdateInput(in *UpdateInput) error {
	if in.Description != nil {
		*in.Description = strings.TrimSpace(*in.Description)
		if err := check.Description(*in.Description); err != nil {
			return err
		}
	}

	return nil
}
