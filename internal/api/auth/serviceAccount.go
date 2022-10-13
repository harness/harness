// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package auth

import (
	"context"

	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types/enum"
)

/*
 * CheckServiceAccount checks if a service account specific permission is granted for the current auth session
 * in the scope of the parent.
 * Returns nil if the permission is granted, otherwise returns an error.
 * NotAuthenticated, NotAuthorized, or any unerlaying error.
 */
func CheckServiceAccount(ctx context.Context, authorizer authz.Authorizer, session *auth.Session,
	spaceStore store.SpaceStore, repoStore store.RepoStore, parentType enum.ParentResourceType, parentID int64,
	saUID string, permission enum.Permission) error {
	return CheckChild(ctx, authorizer, session, spaceStore, repoStore, parentType, parentID,
		enum.ResourceTypeServiceAccount, saUID, permission)
}
