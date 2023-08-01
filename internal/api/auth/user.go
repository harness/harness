// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package auth

import (
	"context"

	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// CheckUser checks if a user specific permission is granted for the current auth session.
// Returns nil if the permission is granted, otherwise returns an error.
// NotAuthenticated, NotAuthorized, or any underlying error.
func CheckUser(ctx context.Context, authorizer authz.Authorizer, session *auth.Session,
	user *types.User, permission enum.Permission,
) error {
	// a user exists outside any scope
	scope := &types.Scope{}
	resource := &types.Resource{
		Type: enum.ResourceTypeUser,
		Name: user.UID,
	}

	return Check(ctx, authorizer, session, scope, resource, permission)
}
