// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"context"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

/*
 * Find tries to find the provided user.
 */
func (c *Controller) Find(ctx context.Context, session *auth.Session,
	userUID string) (*types.User, error) {
	user, err := c.FindNoAuth(ctx, userUID)
	if err != nil {
		return nil, err
	}

	// Ensure principal has required permissions on parent.
	if err = apiauth.CheckUser(ctx, c.authorizer, session, user, enum.PermissionUserView); err != nil {
		return nil, err
	}

	return user, nil
}

/*
 * FindNoAuth finds a user without auth checks.
 * WARNING: Never call as part of user flow.
 */
func (c *Controller) FindNoAuth(ctx context.Context, userUID string) (*types.User, error) {
	return findUserFromUID(ctx, c.userStore, userUID)
}
