// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"

	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ListServiceAccounts lists the service accounts of a repo.

func (c *Controller) ListServiceAccounts(ctx context.Context,
	session *auth.Session,
	repoRef string,
) ([]*types.ServiceAccount, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView, false)
	if err != nil {
		return nil, err
	}

	return c.principalStore.ListServiceAccounts(ctx, enum.ParentResourceTypeRepo, repo.ID)
}
