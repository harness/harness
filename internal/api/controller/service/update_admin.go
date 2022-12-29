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
)

// UpdateAdmin updates the admin state of a service.
func (c *Controller) UpdateAdmin(ctx context.Context, session *auth.Session,
	serviceUID string, admin bool) (*types.Service, error) {
	sbc, err := findServiceFromUID(ctx, c.serviceStore, serviceUID)
	if err != nil {
		return nil, err
	}

	// Ensure principal has required permissions on parent.
	if err = apiauth.CheckService(ctx, c.authorizer, session, sbc, enum.PermissionServiceEditAdmin); err != nil {
		return nil, err
	}

	sbc.Admin = admin
	sbc.Updated = time.Now().UnixMilli()

	err = c.serviceStore.Update(ctx, sbc)
	if err != nil {
		return nil, err
	}

	return sbc, nil
}
