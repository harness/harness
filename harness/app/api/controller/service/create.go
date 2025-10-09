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

package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/dchest/uniuri"
)

// CreateInput is the input used for create operations.
type CreateInput struct {
	UID         string `json:"uid"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

// Create creates a new service.
func (c *Controller) Create(ctx context.Context, session *auth.Session, in *CreateInput) (*types.Service, error) {
	// Ensure principal has required permissions (service is global, no explicit resource)
	scope := &types.Scope{}
	resource := &types.Resource{
		Type: enum.ResourceTypeService,
	}
	if err := apiauth.Check(ctx, c.authorizer, session, scope, resource, enum.PermissionServiceEdit); err != nil {
		return nil, err
	}

	return c.CreateNoAuth(ctx, in, false)
}

/*
 * CreateNoAuth creates a new service without auth checks.
 * WARNING: Never call as part of user flow.
 *
 * Note: take admin separately to avoid potential vulnerabilities for user calls.
 */
func (c *Controller) CreateNoAuth(ctx context.Context, in *CreateInput, admin bool) (*types.Service, error) {
	if err := c.sanitizeCreateInput(in); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	svc := &types.Service{
		UID:         in.UID,
		Email:       in.Email,
		DisplayName: in.DisplayName,
		Admin:       admin,
		Salt:        uniuri.NewLen(uniuri.UUIDLen),
		Created:     time.Now().UnixMilli(),
		Updated:     time.Now().UnixMilli(),
	}

	err := c.principalStore.CreateService(ctx, svc)
	if err != nil {
		return nil, err
	}

	return svc, nil
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
	if err := check.DisplayName(in.DisplayName); err != nil { //nolint:revive
		return err
	}

	return nil
}
