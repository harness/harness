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
	"errors"
	"net/http"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/services/webhook"
	"github.com/harness/gitness/blob"
	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types/check"

	"github.com/rs/zerolog/log"
)

func Translate(err error) *Error {
	var (
		rError      *Error
		checkError  *check.ValidationError
		gitrpcError *gitrpc.Error
		maxBytesErr *http.MaxBytesError
	)

	// TODO: Improve performance of checking multiple errors with errors.Is

	switch {
	// api errors
	case errors.As(err, &rError):
		return rError

	// api auth errors
	case errors.Is(err, apiauth.ErrNotAuthenticated):
		return ErrUnauthorized
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

	//	upload errors
	case errors.Is(err, blob.ErrNotFound):
		return ErrNotFound
	case errors.As(err, &maxBytesErr):
		return RequestTooLargef("The request is too large. maximum allowed size is %d bytes", maxBytesErr.Limit)

	// gitrpc errors
	case errors.As(err, &gitrpcError):
		return NewWithPayload(httpStatusCode(
			gitrpcError.Status),
			gitrpcError.Message,
			gitrpcError.Details,
		)

	// webhook errors
	case errors.Is(err, webhook.ErrWebhookNotRetriggerable):
		return ErrWebhookNotRetriggerable

	// unknown error
	default:
		log.Warn().Msgf("Unable to translate error: %s", err)
		return ErrInternal
	}
}

// lookup of gitrpc error codes to HTTP status codes.
var codes = map[gitrpc.Status]int{
	gitrpc.StatusConflict:           http.StatusConflict,
	gitrpc.StatusInvalidArgument:    http.StatusBadRequest,
	gitrpc.StatusNotFound:           http.StatusNotFound,
	gitrpc.StatusPathNotFound:       http.StatusNotFound,
	gitrpc.StatusNotImplemented:     http.StatusNotImplemented,
	gitrpc.StatusPreconditionFailed: http.StatusPreconditionFailed,
	gitrpc.StatusUnauthorized:       http.StatusUnauthorized,
	gitrpc.StatusInternal:           http.StatusInternalServerError,
	gitrpc.StatusNotMergeable:       http.StatusPreconditionFailed,
}

// httpStatusCode returns the associated HTTP status code for a gitrpc error code.
func httpStatusCode(code gitrpc.Status) int {
	if v, ok := codes[code]; ok {
		return v
	}
	return http.StatusInternalServerError
}
