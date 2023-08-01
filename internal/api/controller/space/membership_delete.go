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

// MembershipDelete removes an existing membership from a space.
func (c *Controller) MembershipDelete(ctx context.Context,
	session *auth.Session,
	spaceRef string,
	userUID string,
) error {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return err
	}

	if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSpaceEdit, false); err != nil {
		return err
	}

	user, err := c.principalStore.FindUserByUID(ctx, userUID)
	if err != nil {
		return fmt.Errorf("failed to find user by uid: %w", err)
	}

	err = c.membershipStore.Delete(ctx, types.MembershipKey{
		SpaceID:     space.ID,
		PrincipalID: user.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete user membership: %w", err)
	}

	return nil
}
