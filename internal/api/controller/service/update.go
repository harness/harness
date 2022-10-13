// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"context"
	"time"

	"github.com/gotidy/ptr"
	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

// UpdateInput store infos to update an existing service.
type UpdateInput struct {
	Name *string `json:"name"`
}

/*
 * Update updates the provided service.
 */
func (c *Controller) Update(ctx context.Context, session *auth.Session,
	serviceUID string, in *UpdateInput) (*types.Service, error) {
	svc, err := findServiceFromUID(ctx, c.serviceStore, serviceUID)
	if err != nil {
		return nil, err
	}

	// Ensure principal has required permissions on parent.
	if err = apiauth.CheckService(ctx, c.authorizer, session, svc, enum.PermissionServiceEdit); err != nil {
		return nil, err
	}

	if in.Name != nil {
		svc.Name = ptr.ToString(in.Name)
	}
	svc.Updated = time.Now().UnixMilli()

	// validate service
	if err = check.Service(svc); err != nil {
		return nil, err
	}

	err = c.serviceStore.Update(ctx, svc)
	if err != nil {
		return nil, err
	}

	return svc, nil
}
