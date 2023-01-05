// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// List lists all services of the system.
func (c *Controller) List(ctx context.Context, session *auth.Session) (int64, []*types.Service, error) {
	// Ensure principal has required permissions (service is global, no explicit resource)
	scope := &types.Scope{}
	resource := &types.Resource{
		Type: enum.ResourceTypeService,
	}
	if err := apiauth.Check(ctx, c.authorizer, session, scope, resource, enum.PermissionServiceView); err != nil {
		return 0, nil, err
	}

	count, err := c.principalStore.CountServices(ctx)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to count services: %w", err)
	}

	repos, err := c.principalStore.ListServices(ctx)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to list services: %w", err)
	}

	return count, repos, nil
}
