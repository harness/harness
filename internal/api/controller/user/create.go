// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"context"
	"fmt"
	"time"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/dchest/uniuri"
	"golang.org/x/crypto/bcrypt"
)

// CreateInput is the input used for create operations.
// On purpose don't expose admin, has to be enabled explicitly.
type CreateInput struct {
	UID         string `json:"uid"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Password    string `json:"password"`
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

	return c.CreateNoAuth(ctx, in, false)
}

/*
 * CreateNoAuth creates a new user without auth checks.
 * WARNING: Never call as part of user flow.
 *
 * Note: take admin separately to avoid potential vulnerabilities for user calls.
 */
func (c *Controller) CreateNoAuth(ctx context.Context, in *CreateInput, admin bool) (*types.User, error) {
	// validate password before hashing
	if err := check.Password(in.Password); err != nil {
		return nil, err
	}

	hash, err := hashPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to create hash: %w", err)
	}

	user := &types.User{
		UID:         in.UID,
		DisplayName: in.DisplayName,
		Email:       in.Email,
		Password:    string(hash),
		Salt:        uniuri.NewLen(uniuri.UUIDLen),
		Created:     time.Now().UnixMilli(),
		Updated:     time.Now().UnixMilli(),
		Admin:       admin,
	}

	// validate user
	if err = c.userCheck(user); err != nil {
		return nil, err
	}

	err = c.userStore.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	// first user will be admin by default.
	// TODO: move responsibility somewhere else.
	if user.ID == 1 {
		user.Admin = true
		err = c.userStore.Update(ctx, user)
		if err != nil {
			return nil, err
		}
	}

	return user, nil
}
