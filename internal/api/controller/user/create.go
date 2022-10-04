// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"context"
	"fmt"
	"time"

	"github.com/dchest/uniuri"
	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
	"golang.org/x/crypto/bcrypt"
)

type CreateInput struct {
	UID      string `json:"uid"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

/*
 * Create creates a new user.
 */
func (c *Controller) Create(ctx context.Context, session *auth.Session, in *CreateInput) (*types.User, error) {
	// Ensure principal has required permissions (user is global, no explicit resource)
	scope := &types.Scope{}
	resource := &types.Resource{
		Type: enum.ResourceTypeUser,
	}
	if err := apiauth.Check(ctx, c.authorizer, session, scope, resource, enum.PermissionUserCreate); err != nil {
		return nil, err
	}

	return c.createNoAuth(ctx, in)
}

/*
 * createNoAuth creates a new user without auth checks.
 */
func (c *Controller) createNoAuth(ctx context.Context, in *CreateInput) (*types.User, error) {
	hash, err := hashPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to create hash: %w", err)
	}

	user := &types.User{
		UID:      in.UID,
		Name:     in.Name,
		Email:    in.Email,
		Password: string(hash),
		Salt:     uniuri.NewLen(uniuri.UUIDLen),
		Created:  time.Now().UnixMilli(),
		Updated:  time.Now().UnixMilli(),
	}

	// validate user
	if err = check.User(user); err != nil {
		return nil, err
	}

	err = c.userStore.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	// first user will be admin by default.
	if user.ID == 1 {
		user.Admin = true
		err = c.userStore.Update(ctx, user)
		if err != nil {
			return nil, err
		}
	}

	return user, nil
}
