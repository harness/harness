// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// MembershipList lists all space memberships.
func (c *Controller) MembershipList(ctx context.Context,
	session *auth.Session,
	spaceRef string,
	opts types.MembershipFilter,
) ([]types.MembershipUser, int64, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, 0, err
	}

	if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSpaceView, false); err != nil {
		return nil, 0, err
	}

	var memberships []types.MembershipUser
	var membershipsCount int64

	err = dbtx.New(c.db).WithTx(ctx, func(ctx context.Context) error {
		memberships, err = c.membershipStore.ListUsers(ctx, space.ID, opts)
		if err != nil {
			return fmt.Errorf("failed to list memberships for space: %w", err)
		}

		membershipsCount, err = c.membershipStore.CountUsers(ctx, space.ID, opts)
		if err != nil {
			return fmt.Errorf("failed to count memberships for space: %w", err)
		}

		return nil
	}, dbtx.TxDefaultReadOnly)
	if err != nil {
		return nil, 0, err
	}

	return memberships, membershipsCount, nil
}
