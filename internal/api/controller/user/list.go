// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

/*
 * List lists all users of the system.
 */
func (c *Controller) List(ctx context.Context, session *auth.Session,
	userFilter *types.UserFilter) (int64, []*types.User, error) {
	// Ensure principal has required permissions (user is global, no explicit resource)
	scope := &types.Scope{}
	resource := &types.Resource{
		Type: enum.ResourceTypeUser,
	}
	if err := apiauth.Check(ctx, c.authorizer, session, scope, resource, enum.PermissionUserView); err != nil {
		return 0, nil, err
	}

	count, err := c.userStore.Count(ctx)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to count users: %w", err)
	}

	repos, err := c.userStore.List(ctx, userFilter)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to list users: %w", err)
	}

	return count, repos, nil
}
