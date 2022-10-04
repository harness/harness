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
 * FindEmail tries to find the provided user using email.
 */
func (c *Controller) FindEmail(ctx context.Context, session *auth.Session,
	email string) (*types.User, error) {
	user, err := findUserFromEmail(ctx, c.userStore, email)
	if err != nil {
		return nil, err
	}

	// Ensure principal has required permissions on parent.
	if err = apiauth.CheckUser(ctx, c.authorizer, session, user, enum.PermissionUserView); err != nil {
		return nil, err
	}

	return user, nil
}
