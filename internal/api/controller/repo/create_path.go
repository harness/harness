// Copyright 2022 Harness Inc. All rights reserved.
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

// CreatePathInput used for path creation apis.
type CreatePathInput struct {
	Path string `json:"path"`
}

// CreatePath creates a new path for a repo.
func (c *Controller) CreatePath(ctx context.Context, session *auth.Session,
	repoRef string, in *CreatePathInput) (*types.Path, error) {
	repo, err := c.repoStore.FindRepoFromRef(ctx, repoRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoEdit, false); err != nil {
		return nil, err
	}

	params := &types.PathParams{
		Path:      in.Path,
		CreatedBy: session.Principal.ID,
		Created:   time.Now().UnixMilli(),
		Updated:   time.Now().UnixMilli(),
	}

	// validate path
	if err = check.PathParams(params, repo.Path, false); err != nil {
		return nil, err
	}

	// TODO: ensure principal is authorized to create a path pointing to in.Path
	path, err := c.repoStore.CreatePath(ctx, repo.ID, params)
	if err != nil {
		return nil, err
	}

	return path, nil
}
