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

package usererror

import (
	"fmt"
	"maps"
	"net/http"
)

var (
	// ErrInternal is returned when an internal error occurred.
	ErrInternal = New(http.StatusInternalServerError, "Internal error occurred")

	// ErrInvalidToken is returned when the api request token is invalid.
	ErrInvalidToken = New(http.StatusUnauthorized, "Invalid or missing token")

	// ErrBadRequest is returned when there was an issue with the input.
	ErrBadRequest = New(http.StatusBadRequest, "Bad Request")

	// ErrUnauthorized is returned when the acting principal is not authenticated.
	ErrUnauthorized = New(http.StatusUnauthorized, "Unauthorized")

	// ErrForbidden is returned when the acting principal is not authorized.
	ErrForbidden = New(http.StatusForbidden, "Forbidden")

	// ErrNotFound is returned when a resource is not found.
	ErrNotFound = New(http.StatusNotFound, "Not Found")

	// ErrPreconditionFailed is returned when a precondition failed.
	ErrPreconditionFailed = New(http.StatusPreconditionFailed, "Precondition failed")

	// ErrNotMergeable is returned when a branch can't be merged.
	ErrNotMergeable = New(http.StatusPreconditionFailed, "Branch can't be merged")

	// ErrNoChange is returned when no change was found based on the request.
	ErrNoChange = New(http.StatusBadRequest, "No Change")

	// ErrDuplicate is returned when a resource already exits.
	ErrDuplicate = New(http.StatusConflict, "Resource already exists")

	// ErrPrimaryPathCantBeDeleted is returned when trying to delete a primary path.
	ErrPrimaryPathCantBeDeleted = New(http.StatusBadRequest, "The primary path of an object can't be deleted")

	// ErrPathTooLong is returned when an action would lead to a path that is too long.
	ErrPathTooLong = New(http.StatusBadRequest, "The resource path is too long")

	// ErrCyclicHierarchy is returned if the action would create a cyclic dependency between spaces.
	ErrCyclicHierarchy = New(http.StatusBadRequest,
		"Unable to perform the action as it would lead to a cyclic dependency")

	// ErrSpaceWithChildsCantBeDeleted is returned if the principal is trying to delete a space that
	// still has child resources.
	ErrSpaceWithChildsCantBeDeleted = New(http.StatusBadRequest,
		"Space can't be deleted as it still contains child resources")

	// ErrDefaultBranchCantBeDeleted is returned if the user tries to delete the default branch of a repository.
	ErrDefaultBranchCantBeDeleted = New(http.StatusBadRequest, "The default branch of a repository can't be deleted")

	// ErrPullReqRefsCantBeModified is returned if a user tries to tinker with a pull request git ref.
	ErrPullReqRefsCantBeModified = New(http.StatusBadRequest, "The pull request git refs can't be modified")

	// ErrRequestTooLarge is returned if the request it too large.
	ErrRequestTooLarge = New(http.StatusRequestEntityTooLarge, "The request is too large")

	// ErrWebhookNotRetriggerable is returned if the webhook can't be retriggered.
	ErrWebhookNotRetriggerable = New(http.StatusMethodNotAllowed,
		"The webhook execution is incomplete and can't be retriggered")

	// ErrCodeOwnersNotFound is returned when codeowners file is not found.
	ErrCodeOwnersNotFound = New(http.StatusNotFound, "CODEOWNERS file not found")

	// ErrResponseNotFlushable is returned if the response writer doesn't implement http.Flusher.
	ErrResponseNotFlushable = New(http.StatusInternalServerError, "Response not streamable")

	// ErrResourceLocked is returned if the resource is locked.
	ErrResourceLocked = New(
		http.StatusLocked,
		"The requested resource is temporarily locked, please retry the operation.",
	)

	// ErrEmptyRepoNeedsBranch is returned if no branch found on the githook post receieve for empty repositories.
	ErrEmptyRepoNeedsBranch = New(http.StatusBadRequest,
		"Pushing to an empty repository requires at least one branch with commits.")

	// ErrGitLFSDisabled is returned if the Git LFS is disabled but LFS endpoint is requested.
	ErrGitLFSDisabled = New(http.StatusBadRequest, "Git LFS is disabled")

	ErrQuarantinedArtifact = New(http.StatusForbidden, "Artifact is quarantined")

	ErrArtifactBlocked = New(http.StatusForbidden, "Artifact is blocked due to policy violations")
)

// Error represents a json-encoded API error.
type Error struct {
	Status  int            `json:"-"`
	Message string         `json:"message"`
	Values  map[string]any `json:"values,omitempty"`
}

func (e *Error) Error() string {
	return e.Message
}

// New returns a new user facing error.
func New(status int, message string) *Error {
	return &Error{Status: status, Message: message}
}

// Newf returns a new user facing error.
func Newf(status int, format string, args ...any) *Error {
	return &Error{Status: status, Message: fmt.Sprintf(format, args...)}
}

// NewWithPayload returns a new user facing error with payload.
func NewWithPayload(status int, message string, valueMaps ...map[string]any) *Error {
	var values map[string]any
	for _, valueMap := range valueMaps {
		if values == nil {
			values = valueMap
			continue
		}
		maps.Copy(values, valueMap)
	}
	return &Error{Status: status, Message: message, Values: values}
}

// BadRequest returns a new user facing bad request error.
func BadRequest(message string) *Error {
	return New(http.StatusBadRequest, message)
}

// BadRequestf returns a new user facing bad request error.
func BadRequestf(format string, args ...any) *Error {
	return Newf(http.StatusBadRequest, format, args...)
}

// RequestTooLargef returns a new user facing request too large error.
func RequestTooLargef(format string, args ...any) *Error {
	return Newf(http.StatusRequestEntityTooLarge, format, args...)
}

// UnprocessableEntity returns a new user facing unprocessable entity error.
func UnprocessableEntity(message string) *Error {
	return New(http.StatusUnprocessableEntity, message)
}

// UnprocessableEntityf returns a new user facing unprocessable entity error.
func UnprocessableEntityf(format string, args ...any) *Error {
	return Newf(http.StatusUnprocessableEntity, format, args...)
}

// BadRequestWithPayload returns a new user facing bad request error with payload.
func BadRequestWithPayload(message string, values ...map[string]any) *Error {
	return NewWithPayload(http.StatusBadRequest, message, values...)
}

// Forbidden returns a new user facing forbidden error.
func Forbidden(message string) *Error {
	return New(http.StatusForbidden, message)
}

func MethodNotAllowed(message string) *Error {
	return New(http.StatusMethodNotAllowed, message)
}

// NotFound returns a new user facing not found error.
func NotFound(message string) *Error {
	return New(http.StatusNotFound, message)
}

// NotFoundf returns a new user facing not found error.
func NotFoundf(format string, args ...any) *Error {
	return Newf(http.StatusNotFound, format, args...)
}

// ConflictWithPayload returns a new user facing conflict error with payload.
func ConflictWithPayload(message string, values ...map[string]any) *Error {
	return NewWithPayload(http.StatusConflict, message, values...)
}

// Conflict returns a new user facing conflict error.
func Conflict(message string) *Error {
	return NewWithPayload(http.StatusConflict, message)
}
