// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package authz

import (
	"context"
	"errors"

	"github.com/harness/gitness/app/auth"
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
