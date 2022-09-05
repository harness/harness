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

func newNotAuthenticatedError(permission enum.Permission, resourceType enum.ResourceType, resourceId string) *notAuthenticatedError {
	return &notAuthenticatedError{
		msg: fmt.Sprintf("Operation %s on %s '%s' requires authentication.", permission, resourceType, resourceId),
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

func newNotAuthorizedError(user *types.User, permission enum.Permission, resourceType enum.ResourceType, resourceId string) *notAuthorizedError {
	return &notAuthorizedError{
		msg: fmt.Sprintf("User '%s' is not authorized to execute %s on %s '%s.", user.Email, permission, resourceType, resourceId),
	}
}

func (e *notAuthorizedError) Error() string {
	return e.msg
}
