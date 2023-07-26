// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// MembershipSpaces lists all spaces in which the user is a member.
func (c *Controller) MembershipSpaces(ctx context.Context,
	session *auth.Session,
	userUID string,
) ([]types.MembershipSpace, error) {
	user, err := findUserFromUID(ctx, c.principalStore, userUID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user by UID: %w", err)
	}

	// Ensure principal has required permissions.
	if err = apiauth.CheckUser(ctx, c.authorizer, session, user, enum.PermissionUserView); err != nil {
		return nil, err
	}

	membershipSpaces, err := c.membershipStore.ListSpaces(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list membership spaces for user: %w", err)
	}

	return membershipSpaces, nil
}
