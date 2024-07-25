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
	"context"
	"net/http"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/controller/limiter"
	"github.com/harness/gitness/app/services/codeowners"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/services/webhook"
	"github.com/harness/gitness/blob"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/lock"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types/check"

	"github.com/rs/zerolog/log"
)

func Translate(ctx context.Context, err error) *Error {
	var (
		rError                   *Error
		checkError               *check.ValidationError
		appError                 *errors.Error
		maxBytesErr              *http.MaxBytesError
		codeOwnersTooLargeError  *codeowners.TooLargeError
		codeOwnersFileParseError *codeowners.FileParseError
		lockError                *lock.Error
	)

	// print original error for debugging purposes
	log.Ctx(ctx).Debug().Err(err).Msgf("translating error to user facing error")

	// TODO: Improve performance of checking multiple errors with errors.Is

	switch {
	// api errors
	case errors.As(err, &rError):
		return rError

	// api auth errors
	case errors.Is(err, apiauth.ErrNotAuthorized):
		return ErrForbidden

	// validation errors
	case errors.As(err, &checkError):
		return New(http.StatusBadRequest, checkError.Error())

	// store errors
	case errors.Is(err, store.ErrResourceNotFound):
		return ErrNotFound
	case errors.Is(err, store.ErrDuplicate):
		return ErrDuplicate
	case errors.Is(err, store.ErrPrimaryPathCantBeDeleted):
		return ErrPrimaryPathCantBeDeleted
	case errors.Is(err, store.ErrPathTooLong):
		return ErrPathTooLong
	case errors.Is(err, store.ErrNoChangeInRequestedMove):
		return ErrNoChange
	case errors.Is(err, store.ErrIllegalMoveCyclicHierarchy):
		return ErrCyclicHierarchy
	case errors.Is(err, store.ErrSpaceWithChildsCantBeDeleted):
		return ErrSpaceWithChildsCantBeDeleted
	case errors.Is(err, limiter.ErrMaxNumReposReached):
		return Forbidden(err.Error())

	//	upload errors
	case errors.Is(err, blob.ErrNotFound):
		return ErrNotFound
	case errors.As(err, &maxBytesErr):
		return RequestTooLargef("The request is too large. maximum allowed size is %d bytes", maxBytesErr.Limit)

	// git errors
	case errors.As(err, &appError):
		if appError.Err != nil {
			log.Ctx(ctx).Warn().Err(appError.Err).Msgf("Application error translation is omitting internal details.")
		}

		return NewWithPayload(
			httpStatusCode(appError.Status),
			appError.Message,
			appError.Details,
		)

	// webhook errors
	case errors.Is(err, webhook.ErrWebhookNotRetriggerable):
		return ErrWebhookNotRetriggerable

	// codeowners errors
	case errors.Is(err, codeowners.ErrNotFound):
		return ErrCodeOwnersNotFound
	case errors.As(err, &codeOwnersTooLargeError):
		return UnprocessableEntityf(codeOwnersTooLargeError.Error())
	case errors.As(err, &codeOwnersFileParseError):
		return NewWithPayload(
			http.StatusUnprocessableEntity,
			codeOwnersFileParseError.Error(),
			map[string]any{
				"line_number": codeOwnersFileParseError.LineNumber,
				"line":        codeOwnersFileParseError.Line,
				"err":         codeOwnersFileParseError.Err.Error(),
			},
		)
	// lock errors
	case errors.As(err, &lockError):
		return errorFromLockError(lockError)

	// public access errors
	case errors.Is(err, publicaccess.ErrPublicAccessNotAllowed):
		return BadRequestf("Public access on resources is not allowed.")

	// unknown error
	default:
		log.Ctx(ctx).Warn().Err(err).Msgf("Unable to translate error - returning Internal Error.")
		return ErrInternal
	}
}

// errorFromLockError returns the associated error for a given lock error.
func errorFromLockError(err *lock.Error) *Error {
	if err.Kind == lock.ErrorKindCannotLock ||
		err.Kind == lock.ErrorKindLockHeld ||
		err.Kind == lock.ErrorKindMaxRetriesExceeded {
		return ErrResourceLocked
	}

	return ErrInternal
}

// lookup of git error codes to HTTP status codes.
var codes = map[errors.Status]int{
	errors.StatusConflict:           http.StatusConflict,
	errors.StatusInvalidArgument:    http.StatusBadRequest,
	errors.StatusNotFound:           http.StatusNotFound,
	errors.StatusNotImplemented:     http.StatusNotImplemented,
	errors.StatusPreconditionFailed: http.StatusPreconditionFailed,
	errors.StatusUnauthorized:       http.StatusUnauthorized,
	errors.StatusInternal:           http.StatusInternalServerError,
}

// httpStatusCode returns the associated HTTP status code for a git error code.
func httpStatusCode(code errors.Status) int {
	if v, ok := codes[code]; ok {
		return v
	}
	return http.StatusInternalServerError
}
