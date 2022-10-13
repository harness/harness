// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"context"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/enum"
)

/*
 * Delete deletes a service.
 */
func (c *Controller) Delete(ctx context.Context, session *auth.Session,
	serviceUID string) error {
	svc, err := findServiceFromUID(ctx, c.serviceStore, serviceUID)
	if err != nil {
		return err
	}

	// Ensure principal has required permissions on parent
	if err = apiauth.CheckService(ctx, c.authorizer, session, svc, enum.PermissionServiceDelete); err != nil {
		return err
	}

	return c.serviceStore.Delete(ctx, svc.ID)
}
