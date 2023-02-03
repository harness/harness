// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"context"
	"fmt"
	"strings"
	"time"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"golang.org/x/crypto/bcrypt"
)

// UpdateInput store infos to update an existing user.
type UpdateInput struct {
	Email       *string `json:"email"`
	Password    *string `json:"password"`
	DisplayName *string `json:"display_name"`
}

// Update updates the provided user.
func (c *Controller) Update(ctx context.Context, session *auth.Session,
	userUID string, in *UpdateInput) (*types.User, error) {
	user, err := findUserFromUID(ctx, c.principalStore, userUID)
	if err != nil {
		return nil, err
	}

	// Ensure principal has required permissions on parent.
	if err = apiauth.CheckUser(ctx, c.authorizer, session, user, enum.PermissionUserEdit); err != nil {
		return nil, err
	}

	if err = c.sanitizeUpdateInput(in); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	if in.DisplayName != nil {
		user.DisplayName = *in.DisplayName
	}
	if in.Email != nil {
		user.Email = *in.Email
	}
	if in.Password != nil {
		var hash []byte
		hash, err = hashPassword([]byte(*in.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		user.Password = string(hash)
	}
	user.Updated = time.Now().UnixMilli()

	err = c.principalStore.UpdateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (c *Controller) sanitizeUpdateInput(in *UpdateInput) error {
	if in.Email != nil {
		*in.Email = strings.TrimSpace(*in.Email)
		if err := check.Email(*in.Email); err != nil {
			return err
		}
	}

	if in.DisplayName != nil {
		*in.DisplayName = strings.TrimSpace(*in.DisplayName)
		if err := check.DisplayName(*in.DisplayName); err != nil {
			return err
		}
	}

	if in.Password != nil {
		if err := check.Password(*in.Password); err != nil {
			return err
		}
	}

	return nil
}
