// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

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
	if err := check.DisplayName(in.DisplayName); err != nil {
		return err
	}

	return nil
}
