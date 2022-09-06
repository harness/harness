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
	msg string
}

func IsNotAuthenticatedError(err error) bool {
	_, ok := err.(*notAuthenticatedError)
	return ok
}

func newNotAuthenticatedError(permission enum.Permission, resource *types.Resource) *notAuthenticatedError {
	return &notAuthenticatedError{
		msg: fmt.Sprintf("Operation %s on resource %v requires authentication.", permission, resource),
	}
}

func (e *notAuthenticatedError) Error() string {
	return e.msg
}

type notAuthorizedError struct {
	msg string
}

func IsNotAuthorizedError(err error) bool {
	_, ok := err.(*notAuthorizedError)
	return ok
}

func newNotAuthorizedError(user *types.User, scope *types.Scope, resource *types.Resource, permission enum.Permission) *notAuthorizedError {
	return &notAuthorizedError{
		msg: fmt.Sprintf(
			"User '%s' (%s) is not authorized to execute %s on resource %v in scope %v.",
			user.Name,
			user.Email,
			permission,
			resource,
			scope),
	}
}

func (e *notAuthorizedError) Error() string {
	return e.msg
}
