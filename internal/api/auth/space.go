// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package auth

import (
	"context"

	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/paths"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/pkg/errors"
)

/*
 * CheckSpace checks if a space specific permission is granted for the current auth session
 * in the scope of its parent.
 * Returns nil if the permission is granted, otherwise returns an error.
 * NotAuthenticated, NotAuthorized, or any unerlaying error.
 */
func CheckSpace(ctx context.Context, authorizer authz.Authorizer, session *auth.Session,
	space *types.Space, permission enum.Permission, orPublic bool) error {
	if orPublic && space.IsPublic {
		return nil
	}

	parentSpace, name, err := paths.DisectLeaf(space.Path)
	if err != nil {
		return errors.Wrapf(err, "Failed to disect path '%s'", space.Path)
	}

	scope := &types.Scope{SpacePath: parentSpace}
	resource := &types.Resource{
		Type: enum.ResourceTypeSpace,
		Name: name,
	}

	return Check(ctx, authorizer, session, scope, resource, permission)
}
