// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// Delete deletes a user.
func (c *Controller) Delete(ctx context.Context, session *auth.Session,
	userUID string) error {
	user, err := findUserFromUID(ctx, c.principalStore, userUID)
	if err != nil {
		return err
	}

	// Fail if the user being deleted is the only admin in DB
	if user.Admin {
		admUsrCount, err := c.principalStore.CountUsers(ctx, &types.UserFilter{Admin: true})
		if err != nil {
			return fmt.Errorf("failed to check admin user count: %w", err)
		}

		if admUsrCount == 1 {
			return usererror.BadRequest("cannot delete the only admin user")
		}
	}

	// Ensure principal has required permissions on parent
	if err = apiauth.CheckUser(ctx, c.authorizer, session, user, enum.PermissionUserDelete); err != nil {
		return err
	}

	// delete all tokens (okay if we fail after - user intended to be deleted anyway)
	// TODO: cascading delete?
	err = c.tokenStore.DeleteForPrincipal(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("failed to delete tokens for user: %w", err)
	}

	return c.principalStore.DeleteUser(ctx, user.ID)
}
