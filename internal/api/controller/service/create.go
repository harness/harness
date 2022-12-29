// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"context"
	"time"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
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
	if err := apiauth.Check(ctx, c.authorizer, session, scope, resource, enum.PermissionServiceCreate); err != nil {
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
	svc := &types.Service{
		UID:         in.UID,
		Email:       in.Email,
		DisplayName: in.DisplayName,
		Admin:       admin,
		Salt:        uniuri.NewLen(uniuri.UUIDLen),
		Created:     time.Now().UnixMilli(),
		Updated:     time.Now().UnixMilli(),
	}

	// validate service
	if err := c.serviceCheck(svc); err != nil {
		return nil, err
	}

	err := c.serviceStore.Create(ctx, svc)
	if err != nil {
		return nil, err
	}

	return svc, nil
}
