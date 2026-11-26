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
	stderrors "errors"
	"fmt"
	"net/http"
	"testing"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/services/codeowners"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/services/webhook"
	"github.com/harness/gitness/blob"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/lock"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types/check"

	"github.com/stretchr/testify/require"
)

func TestTranslate_Table(t *testing.T) {
	maxBytes := &http.MaxBytesError{Limit: 10}

	tests := []struct {
		name    string
		err     error
		status  int
		message string
	}{
		{"user error passes through", ErrBadRequest, http.StatusBadRequest, "Bad Request"},
		{"wrapped user error", fmt.Errorf("ctx: %w", ErrNotFound), http.StatusNotFound, "Not Found"},
		{"api forbidden", apiauth.ErrForbidden, http.StatusForbidden, ErrForbidden.Message},
		{"api unauthorized", apiauth.ErrUnauthorized, http.StatusUnauthorized, ErrUnauthorized.Message},
		{"validation", check.NewValidationError("bad name"), http.StatusBadRequest, "bad name"},
		{"not found", store.ErrResourceNotFound, http.StatusNotFound, ErrNotFound.Message},
		{"duplicate", store.ErrDuplicate, http.StatusConflict, ErrDuplicate.Message},
		{"primary path", store.ErrPrimaryPathCantBeDeleted, http.StatusBadRequest, ErrPrimaryPathCantBeDeleted.Message},
		{"path too long", store.ErrPathTooLong, http.StatusBadRequest, ErrPathTooLong.Message},
		{"no change", store.ErrNoChangeInRequestedMove, http.StatusBadRequest, ErrNoChange.Message},
		{"cyclic", store.ErrIllegalMoveCyclicHierarchy, http.StatusBadRequest, ErrCyclicHierarchy.Message},
		{"space with children", store.ErrSpaceWithChildsCantBeDeleted, http.StatusBadRequest,
			ErrSpaceWithChildsCantBeDeleted.Message},
		{"blob not found", blob.ErrNotFound, http.StatusNotFound, ErrNotFound.Message},
		{"too large", maxBytes, http.StatusRequestEntityTooLarge,
			"The request is too large. maximum allowed size is 10 bytes"},
		{"license expired", store.ErrLicenseExpired, http.StatusBadRequest, "license expired."},
		{"license not found", store.ErrLicenseNotFound, http.StatusBadRequest, "license not found."},
		{"webhook not retriggerable", webhook.ErrWebhookNotRetriggerable, http.StatusMethodNotAllowed,
			ErrWebhookNotRetriggerable.Message},
		{"codeowners not found", codeowners.ErrNotFound, http.StatusNotFound, ErrCodeOwnersNotFound.Message},
		{"codeowners too large", &codeowners.TooLargeError{FileSize: 100}, http.StatusUnprocessableEntity,
			(&codeowners.TooLargeError{FileSize: 100}).Error()},
		{"codeowners invalid type", &codeowners.InvalidFileTypeError{}, http.StatusUnprocessableEntity,
			(&codeowners.InvalidFileTypeError{}).Error()},
		{"public access", publicaccess.ErrPublicAccessNotAllowed, http.StatusBadRequest,
			"Public access on resources is not allowed."},
		{"unknown", stderrors.New("boom"), http.StatusInternalServerError, ErrInternal.Message},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Translate(context.Background(), tt.err)
			require.Equal(t, tt.status, got.Status)
			require.Equal(t, tt.message, got.Message)
		})
	}
}

func TestTranslate_CodeOwnersFileParseError(t *testing.T) {
	got := Translate(context.Background(), &codeowners.FileParseError{
		LineNumber: 3, Line: "bad line", Err: stderrors.New("no owner"),
	})

	require.Equal(t, http.StatusUnprocessableEntity, got.Status)
	require.Equal(t, int64(3), got.Values["line_number"])
	require.Equal(t, "bad line", got.Values["line"])
	require.Equal(t, "no owner", got.Values["err"])
}

func TestTranslate_UnrelatedHistories(t *testing.T) {
	got := Translate(context.Background(), &api.UnrelatedHistoriesError{BaseRef: "main", HeadRef: "dev"})

	require.Equal(t, http.StatusBadRequest, got.Status)
	require.Equal(t, map[string]any{"base_ref": "main", "head_ref": "dev"}, got.Values)
}

func TestTranslate_AppError(t *testing.T) {
	tests := []struct {
		name   string
		err    *errors.Error
		status int
	}{
		{"conflict", errors.Conflict("c"), http.StatusConflict},
		{"invalid argument", errors.InvalidArgument("i"), http.StatusBadRequest},
		{"not found", errors.NotFound("n"), http.StatusNotFound},
		{"internal", errors.Internal(stderrors.New("db down"), "oops"), http.StatusInternalServerError},
		{"unknown status", errors.Format(errors.Status("weird"), "w"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Translate(context.Background(), tt.err)
			require.Equal(t, tt.status, got.Status)
			// Internal details of the wrapped error must not leak.
			require.NotContains(t, got.Message, "db down")
		})
	}

	withDetails := Translate(context.Background(), errors.InvalidArgument("x").SetDetails(map[string]any{"k": "v"}))
	require.Equal(t, map[string]any{"k": "v"}, withDetails.Values)
}

func TestTranslate_LockErrors(t *testing.T) {
	for _, kind := range []lock.ErrorKind{
		lock.ErrorKindCannotLock, lock.ErrorKindLockHeld, lock.ErrorKindMaxRetriesExceeded,
	} {
		got := Translate(context.Background(), lock.NewError(kind, "key", nil))
		require.Equal(t, ErrResourceLocked, got, string(kind))
	}

	for _, kind := range []lock.ErrorKind{
		lock.ErrorKindProviderError, lock.ErrorKindContext, lock.ErrorKindLockNotHeld,
	} {
		got := Translate(context.Background(), lock.NewError(kind, "key", nil))
		require.Equal(t, ErrInternal, got, string(kind))
	}
}

func TestHTTPStatusCode(t *testing.T) {
	require.Equal(t, http.StatusNotImplemented, httpStatusCode(errors.StatusNotImplemented))
	require.Equal(t, http.StatusPreconditionFailed, httpStatusCode(errors.StatusPreconditionFailed))
	require.Equal(t, http.StatusUnauthorized, httpStatusCode(errors.StatusUnauthorized))
	require.Equal(t, http.StatusForbidden, httpStatusCode(errors.StatusForbidden))
	require.Equal(t, http.StatusInternalServerError, httpStatusCode(errors.Status("unknown")))
}

func TestConstructors(t *testing.T) {
	tests := []struct {
		name    string
		got     *Error
		status  int
		message string
	}{
		{"New", New(http.StatusTeapot, "tea"), http.StatusTeapot, "tea"},
		{"Newf", Newf(http.StatusTeapot, "tea %d", 2), http.StatusTeapot, "tea 2"},
		{"BadRequest", BadRequest("b"), http.StatusBadRequest, "b"},
		{"BadRequestf", BadRequestf("b%d", 1), http.StatusBadRequest, "b1"},
		{"RequestTooLargef", RequestTooLargef("max %d", 5), http.StatusRequestEntityTooLarge, "max 5"},
		{"UnprocessableEntity", UnprocessableEntity("u"), http.StatusUnprocessableEntity, "u"},
		{"UnprocessableEntityf", UnprocessableEntityf("u%d", 1), http.StatusUnprocessableEntity, "u1"},
		{"Forbidden", Forbidden("f"), http.StatusForbidden, "f"},
		{"MethodNotAllowed", MethodNotAllowed("m"), http.StatusMethodNotAllowed, "m"},
		{"NotFound", NotFound("n"), http.StatusNotFound, "n"},
		{"NotFoundf", NotFoundf("n%d", 1), http.StatusNotFound, "n1"},
		{"Conflict", Conflict("c"), http.StatusConflict, "c"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.status, tt.got.Status)
			require.Equal(t, tt.message, tt.got.Message)
			require.Equal(t, tt.message, tt.got.Error())
			require.Empty(t, tt.got.Values)
		})
	}
}

func TestPayloadConstructors(t *testing.T) {
	merged := NewWithPayload(http.StatusBadRequest, "m", map[string]any{"a": 1}, map[string]any{"b": 2})
	require.Equal(t, map[string]any{"a": 1, "b": 2}, merged.Values)

	require.Nil(t, NewWithPayload(http.StatusBadRequest, "m").Values)

	bad := BadRequestWithPayload("bad", map[string]any{"a": 1})
	require.Equal(t, http.StatusBadRequest, bad.Status)
	require.Equal(t, map[string]any{"a": 1}, bad.Values)

	conflict := ConflictWithPayload("dup", map[string]any{"id": 7})
	require.Equal(t, http.StatusConflict, conflict.Status)
	require.Equal(t, map[string]any{"id": 7}, conflict.Values)
}
