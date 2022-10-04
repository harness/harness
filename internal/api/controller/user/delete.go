// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/enum"
)

/*
 * Delete deletes a user.
 */
func (c *Controller) Delete(ctx context.Context, session *auth.Session,
	userUID string) error {
	user, err := findUserFromUID(ctx, c.userStore, userUID)
	if err != nil {
		return err
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

	return c.userStore.Delete(ctx, user.ID)
}
