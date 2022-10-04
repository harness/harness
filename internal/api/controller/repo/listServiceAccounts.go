// Copyright 2021 Harness Inc. All rights reserved.
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

/*
* ListServiceAccounts lists the service accounts of a repo.
 */
func (c *Controller) ListServiceAccounts(ctx context.Context, session *auth.Session,
	repoRef string) ([]*types.ServiceAccount, error) {
	repo, err := findRepoFromRef(ctx, c.repoStore, repoRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoView, false); err != nil {
		return nil, err
	}

	return c.saStore.List(ctx, enum.ParentResourceTypeRepo, repo.ID)
}
