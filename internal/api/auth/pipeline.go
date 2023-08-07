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

// CheckRepo checks if a repo specific permission is granted for the current auth session
// in the scope of its parent.
// Returns nil if the permission is granted, otherwise returns an error.
// NotAuthenticated, NotAuthorized, or any underlying error.
func CheckPipeline(ctx context.Context, authorizer authz.Authorizer, session *auth.Session,
	parentPath, uid string, permission enum.Permission) error {
	scope := &types.Scope{SpacePath: parentPath}
	resource := &types.Resource{
		Type: enum.ResourceTypeRepo,
		Name: uid,
	}

	return Check(ctx, authorizer, session, scope, resource, permission)
}
