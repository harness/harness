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

	"github.com/gotidy/ptr"
)

// UpdateInput store infos to update an existing service.
type UpdateInput struct {
	Email       *string `json:"email"`
	DisplayName *string `json:"display_name"`
}

// Update updates the provided service.
func (c *Controller) Update(ctx context.Context, session *auth.Session,
	serviceUID string, in *UpdateInput) (*types.Service, error) {
	svc, err := findServiceFromUID(ctx, c.principalStore, serviceUID)
	if err != nil {
		return nil, err
	}

	// Ensure principal has required permissions on parent.
	if err = apiauth.CheckService(ctx, c.authorizer, session, svc, enum.PermissionServiceEdit); err != nil {
		return nil, err
	}

	if err = c.sanitizeUpdateInput(in); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	if in.Email != nil {
		svc.DisplayName = ptr.ToString(in.Email)
	}
	if in.DisplayName != nil {
		svc.DisplayName = ptr.ToString(in.DisplayName)
	}
	svc.Updated = time.Now().UnixMilli()

	err = c.principalStore.UpdateService(ctx, svc)
	if err != nil {
		return nil, err
	}

	return svc, nil
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

	return nil
}
