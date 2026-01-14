// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package user

import (
	"context"
	"fmt"
	"strings"
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	userevents "github.com/harness/gitness/app/events/user"
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
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

// Create creates a new user.
func (c *Controller) Create(ctx context.Context, session *auth.Session, in *CreateInput) (*types.User, error) {
	// Ensure principal has required permissions (user is global, no explicit resource)
	scope := &types.Scope{}
	resource := &types.Resource{
		Type: enum.ResourceTypeUser,
	}
	if err := apiauth.Check(ctx, c.authorizer, session, scope, resource, enum.PermissionUserEdit); err != nil {
		return nil, err
	}

	user, err := c.CreateNoAuth(ctx, in, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	c.eventReporter.Created(ctx, &userevents.CreatedPayload{
		Base: userevents.Base{
			PrincipalID: session.Principal.ID,
		},
		CreatedPrincipalID: user.ID,
	})

	return user, nil
}

// CreateNoAuth creates a new user without auth checks.
// WARNING: Never call as part of user flow.
//
// Note: take admin separately to avoid potential vulnerabilities for user calls.
func (c *Controller) CreateNoAuth(ctx context.Context, in *CreateInput, admin bool) (*types.User, error) {
	if err := c.sanitizeCreateInput(in); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
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

	err = c.principalStore.CreateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	uCount, err := c.principalStore.CountUsers(ctx, &types.UserFilter{})
	if err != nil {
		return nil, err
	}

	// first 'user' principal will be admin by default.
	if uCount == 1 {
		user.Admin = true
		err = c.principalStore.UpdateUser(ctx, user)
		if err != nil {
			return nil, err
		}
	}

	return user, nil
}

func (c *Controller) sanitizeCreateInput(in *CreateInput) error {
	if err := c.principalUIDCheck(in.UID); err != nil {
		return err
	}

	in.Email = strings.TrimSpace(in.Email)
	if err := check.Email(in.Email); err != nil {
		return err
	}

	in.DisplayName = strings.TrimSpace(in.DisplayName)
	if err := check.DisplayName(in.DisplayName); err != nil {
		return err
	}

	//nolint:revive
	if err := check.Password(in.Password); err != nil {
		return err
	}

	return nil
}
