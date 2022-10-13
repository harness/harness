// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"context"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

/*
 * Find tries to find the provided service.
 */
func (c *Controller) Find(ctx context.Context, session *auth.Session,
	serviceUID string) (*types.Service, error) {
	svc, err := c.FindNoAuth(ctx, serviceUID)
	if err != nil {
		return nil, err
	}

	// Ensure principal has required permissions on parent.
	if err = apiauth.CheckService(ctx, c.authorizer, session, svc, enum.PermissionServiceView); err != nil {
		return nil, err
	}

	return svc, nil
}

/*
 * FindNoAuth finds a service without auth checks.
 * WARNING: Never call as part of user flow.
 */
func (c *Controller) FindNoAuth(ctx context.Context, serviceUID string) (*types.Service, error) {
	return findServiceFromUID(ctx, c.serviceStore, serviceUID)
}
