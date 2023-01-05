// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"context"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

/*
* ListServiceAccounts lists the service accounts of a space.
 */
func (c *Controller) ListServiceAccounts(ctx context.Context, session *auth.Session,
	spaceRef string) ([]*types.ServiceAccount, error) {
	space, err := c.spaceStore.FindSpaceFromRef(ctx, spaceRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSpaceView, false); err != nil {
		return nil, err
	}

	return c.principalStore.ListServiceAccounts(ctx, enum.ParentResourceTypeSpace, space.ID)
}
