// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// MoveInput is used for moving a repo.
type MoveInput struct {
	UID *string `json:"uid"`
}

func (i *MoveInput) hasChanges(repo *types.Repository) bool {
	if i.UID != nil && *i.UID != repo.UID {
		return true
	}

	return false
}

// Move moves a repository to a new space uid.
// TODO: Add support for moving to other parents and aliases.
//
//nolint:gocognit // refactor if needed
func (c *Controller) Move(ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *MoveInput,
) (*types.Repository, error) {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, err
	}

	if repo.Importing {
		return nil, usererror.BadRequest("can't move a repo that is being imported")
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoEdit, false); err != nil {
		return nil, err
	}

	if !in.hasChanges(repo) {
		return repo, nil
	}

	if err = c.sanitizeMoveInput(in); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	repo, err = c.repoStore.UpdateOptLock(ctx, repo, func(r *types.Repository) error {
		if in.UID != nil {
			r.UID = *in.UID
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update repo: %w", err)
	}

	repo.GitURL = c.urlProvider.GenerateRepoCloneURL(repo.Path)

	return repo, nil
}

func (c *Controller) sanitizeMoveInput(in *MoveInput) error {
	if in.UID != nil {
		if err := c.uidCheck(*in.UID, false); err != nil {
			return err
		}
	}

	return nil
}
