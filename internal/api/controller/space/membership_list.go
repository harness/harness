// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// MembershipList lists all space memberships.
func (c *Controller) MembershipList(ctx context.Context,
	session *auth.Session,
	spaceRef string,
) ([]*types.Membership, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSpaceView, false); err != nil {
		return nil, err
	}

	memberships, err := c.membershipStore.ListForSpace(ctx, space.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list memberships for space: %w", err)
	}

	return memberships, nil
}
