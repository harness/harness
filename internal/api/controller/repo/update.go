// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"time"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

// UpdateInput is used for updating a repo.
type UpdateInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	IsPublic    *bool   `json:"isPublic"`
}

/*
* Update updates a repository.
 */
func (c *Controller) Update(ctx context.Context, session *auth.Session,
	repoRef string, in *UpdateInput) (*types.Repository, error) {
	repo, err := findRepoFromRef(ctx, c.repoStore, repoRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoEdit, false); err != nil {
		return nil, err
	}

	// update values only if provided
	if in.Name != nil {
		repo.Name = *in.Name
	}
	if in.Description != nil {
		repo.Description = *in.Description
	}
	if in.IsPublic != nil {
		repo.IsPublic = *in.IsPublic
	}

	// always update time
	repo.Updated = time.Now().UnixMilli()

	// ensure provided values are valid
	if err = check.Repo(repo); err != nil {
		return nil, err
	}

	err = c.repoStore.Update(ctx, repo)
	if err != nil {
		return nil, err
	}

	return repo, nil
}
