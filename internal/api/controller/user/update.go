// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"context"
	"fmt"
	"time"

	"github.com/gotidy/ptr"
	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
	"golang.org/x/crypto/bcrypt"
)

// UpdateInput store infos to update an existing user.
type UpdateInput struct {
	Email       *string `json:"email"`
	Password    *string `json:"password"`
	DisplayName *string `json:"displayName"`
}

/*
 * Update updates the provided user.
 */
func (c *Controller) Update(ctx context.Context, session *auth.Session,
	userUID string, in *UpdateInput) (*types.User, error) {
	user, err := findUserFromUID(ctx, c.userStore, userUID)
	if err != nil {
		return nil, err
	}

	// Ensure principal has required permissions on parent.
	if err = apiauth.CheckUser(ctx, c.authorizer, session, user, enum.PermissionUserEdit); err != nil {
		return nil, err
	}

	if in.DisplayName != nil {
		user.DisplayName = ptr.ToString(in.DisplayName)
	}
	if in.Email != nil {
		user.Email = ptr.ToString(in.Email)
	}
	if in.Password != nil {
		var hash []byte
		hash, err = hashPassword([]byte(ptr.ToString(in.Password)), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		user.Password = string(hash)
	}
	user.Updated = time.Now().UnixMilli()

	// validate user
	if err = c.userCheck(user); err != nil {
		return nil, err
	}

	err = c.userStore.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	return user, nil
}
