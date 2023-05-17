// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package authz

import (
	"context"
	"errors"

	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

var (
	// ErrNoPermissionCheckProvided is error that is thrown if no permission checks are provided.
	ErrNoPermissionCheckProvided = errors.New("no permission checks provided")
)

// Authorizer abstraction of an entity responsible for authorizing access to resources.
type Authorizer interface {
	/*
	 * Checks whether the principal of the current session with the provided metadata
	 * has the permission to execute the action on the resource within the scope.
	 * Returns
	 *		(true, nil)   - the action is permitted
	 *		(false, nil)  - the action is not permitted
	 *		(false, err)  - an error occurred while performing the permission check and the action should be denied
	 */
	Check(ctx context.Context,
		session *auth.Session,
		scope *types.Scope,
		resource *types.Resource,
		permission enum.Permission) (bool, error)

	/*
	 * Checks whether the principal of the current session with the provided metadata
	 * has the permission to execute ALL the action on the resource within the scope.
	 * Returns
	 *		(true, nil)   - all requested actions are permitted
	 *		(false, nil)  - at least one requested action is not permitted
	 *		(false, err)  - an error occurred while performing the permission check and all actions should be denied
	 */
	CheckAll(ctx context.Context,
		session *auth.Session,
		permissionChecks ...types.PermissionCheck) (bool, error)
}
