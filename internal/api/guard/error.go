// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package guard

import (
	"fmt"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type notAuthenticatedError struct {
	resource   *types.Resource
	permission enum.Permission
}

func (e *notAuthenticatedError) Error() string {
	return fmt.Sprintf("Operation %s on resource %v requires authentication.", e.permission, e.resource)
}

func (e *notAuthenticatedError) Is(target error) bool {
	_, ok := target.(*notAuthenticatedError)
	return ok
}

type notAuthorizedError struct {
	user       *types.User
	scope      *types.Scope
	resource   *types.Resource
	permission enum.Permission
}

func (e *notAuthorizedError) Error() string {
	// ASSUMPTION: user is never nil at this point (type is not exported)
	return fmt.Sprintf(
		"User '%s' (%s) is not authorized to execute %s on resource %v in scope %v.",
		e.user.Name,
		e.user.Email,
		e.permission,
		e.resource,
		e.scope)
}

func (e *notAuthorizedError) Is(target error) bool {
	_, ok := target.(*notAuthorizedError)
	return ok
}
