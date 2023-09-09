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

// Find finds a repo.
func (c *Controller) Find(ctx context.Context, session *auth.Session, repoRef string) (*types.Repository, error) {
	// note: can't use c.getRepoCheckAccess because even repositories that are currently being imported can be fetched.
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoView, true); err != nil {
		return nil, err
	}

	// backfill clone url
	repo.GitURL = c.urlProvider.GenerateRepoCloneURL(repo.Path)

	return repo, nil
}
