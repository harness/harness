// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package render

var (
	// ErrInternal is returned when an internal error occured.
	ErrInternal = New("Internal error occured")

	// ErrInvalidToken is returned when the api request token is invalid.
	ErrInvalidToken = New("Invalid or missing token")

	// ErrBadRequest is returned when there was an issue with the user input.
	ErrBadRequest = New("Bad Request")

	// ErrUnauthorized is returned when the user is not authorized.
	ErrUnauthorized = New("Unauthorized")

	// ErrForbidden is returned when user access is forbidden.
	ErrForbidden = New("Forbidden")

	// ErrNotFound is returned when a resource is not found.
	ErrNotFound = New("Not Found")

	// ErrNoChange is returned when no change was found based on the request.
	ErrNoChange = New("No Change")

	// ErrDuplicate is returned when a resource already exits.
	ErrDuplicate = New("Resource already exists")

	// ErrPrimaryPathCantBeDeleted is returned when the user is trying to delete a primary path.
	ErrPrimaryPathCantBeDeleted = New("The primary path of an object can't be deleted")

	// ErrPathTooLong is returned if user action would lead to a path that is too long.
	ErrPathTooLong = New("The resource path is too long")

	// ErrCyclicHierarchy is returned if the user action would create a cyclic dependency between spaces.
	ErrCyclicHierarchy = New("Unable to perform the action as it would lead to a cyclic dependency.")

	// ErrSpaceWithChildsCantBeDeleted is returned if the user is trying to delete a space that
	// still has child resources.
	ErrSpaceWithChildsCantBeDeleted = New("Space can't be deleted as it still contains child resources.")
)

// Error represents a json-encoded API error.
type Error struct {
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return e.Message
}

// New returns a new error message.
func New(text string) *Error {
	return &Error{Message: text}
}
