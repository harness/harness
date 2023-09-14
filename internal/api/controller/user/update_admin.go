// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"context"
	"fmt"
	"time"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type UpdateAdminInput struct {
	Admin bool `json:"admin"`
}

// UpdateAdmin updates the admin state of a user.
func (c *Controller) UpdateAdmin(ctx context.Context, session *auth.Session,
	userUID string, request *UpdateAdminInput) (*types.User, error) {
	user, err := findUserFromUID(ctx, c.principalStore, userUID)
	if err != nil {
		return nil, err
	}

	// Ensure principal has required permissions on parent.
	if err = apiauth.CheckUser(ctx, c.authorizer, session, user, enum.PermissionUserEditAdmin); err != nil {
		return nil, err
	}

	// Fail if the user being updated is the only admin in DB.
	if request.Admin == false && user.Admin == true {
		admUsrCount, err := c.principalStore.CountUsers(ctx, &types.UserFilter{Admin: true})
		if err != nil {
			return nil, fmt.Errorf("failed to check admin user count: %w", err)
		}

		if admUsrCount == 1 {
			return nil, usererror.BadRequest("system requires at least one admin user")
		}

		return user, nil
	}
	user.Admin = request.Admin
	user.Updated = time.Now().UnixMilli()

	err = c.principalStore.UpdateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	return user, nil
}
