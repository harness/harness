// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package usererror

import "net/http"

var (
	// ErrInternal is returned when an internal error occured.
	ErrInternal = New(http.StatusInternalServerError, "Internal error occured")

	// ErrInvalidToken is returned when the api request token is invalid.
	ErrInvalidToken = New(http.StatusUnauthorized, "Invalid or missing token")

	// ErrBadRequest is returned when there was an issue with the input.
	ErrBadRequest = New(http.StatusBadGateway, "Bad Request")

	// ErrUnauthorized is returned when the acting principal is not authenticated.
	ErrUnauthorized = New(http.StatusUnauthorized, "Unauthorized")

	// ErrForbidden is returned when the acting principal is not authorized.
	ErrForbidden = New(http.StatusForbidden, "Forbidden")

	// ErrNotFound is returned when a resource is not found.
	ErrNotFound = New(http.StatusNotFound, "Not Found")

	// ErrNoChange is returned when no change was found based on the request.
	ErrNoChange = New(http.StatusBadRequest, "No Change")

	// ErrDuplicate is returned when a resource already exits.
	ErrDuplicate = New(http.StatusBadRequest, "Resource already exists")

	// ErrPrimaryPathCantBeDeleted is returned when trying to delete a primary path.
	ErrPrimaryPathCantBeDeleted = New(http.StatusBadRequest, "The primary path of an object can't be deleted")

	// ErrPathTooLong is returned when an action would lead to a path that is too long.
	ErrPathTooLong = New(http.StatusBadRequest, "The resource path is too long")

	// ErrCyclicHierarchy is returned if the action would create a cyclic dependency between spaces.
	ErrCyclicHierarchy = New(http.StatusBadRequest, "Unable to perform the action as it would lead to a cyclic dependency")

	// ErrSpaceWithChildsCantBeDeleted is returned if the principal is trying to delete a space that
	// still has child resources.
	ErrSpaceWithChildsCantBeDeleted = New(http.StatusBadRequest,
		"Space can't be deleted as it still contains child resources")
)

// Error represents a json-encoded API error.
type Error struct {
	Status  int    `json:"-"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return e.Message
}

// New returns a new error message.
func New(status int, message string) *Error {
	return &Error{Status: status, Message: message}
}

// New returns a new error message.
func BadRequest(message string) *Error {
	return New(http.StatusBadRequest, message)
}
