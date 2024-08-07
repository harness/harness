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

package render

import (
	"context"
	"net/http"

	"github.com/harness/gitness/app/api/usererror"

	"github.com/rs/zerolog/log"
)

// TranslatedUserError writes the translated user error of the provided error.
func TranslatedUserError(ctx context.Context, w http.ResponseWriter, err error) {
	UserError(ctx, w, usererror.Translate(ctx, err))
}

// NotFound writes the json-encoded message for a not found error.
func NotFound(ctx context.Context, w http.ResponseWriter) {
	UserError(ctx, w, usererror.ErrNotFound)
}

// Unauthorized writes the json-encoded message for an unauthorized error.
func Unauthorized(ctx context.Context, w http.ResponseWriter) {
	UserError(ctx, w, usererror.ErrUnauthorized)
}

// Forbidden writes the json-encoded message for a forbidden error.
func Forbidden(ctx context.Context, w http.ResponseWriter) {
	UserError(ctx, w, usererror.ErrForbidden)
}

// Forbiddenf writes the json-encoded message with a forbidden error.
func Forbiddenf(ctx context.Context, w http.ResponseWriter, format string, args ...interface{}) {
	UserError(ctx, w, usererror.Newf(http.StatusForbidden, format, args...))
}

// BadRequest writes the json-encoded message for a bad request error.
func BadRequest(ctx context.Context, w http.ResponseWriter) {
	UserError(ctx, w, usererror.ErrBadRequest)
}

// BadRequestf writes the json-encoded message with a bad request status code.
func BadRequestf(ctx context.Context, w http.ResponseWriter, format string, args ...interface{}) {
	UserError(ctx, w, usererror.Newf(http.StatusBadRequest, format, args...))
}

// InternalError writes the json-encoded message for an internal error.
func InternalError(ctx context.Context, w http.ResponseWriter) {
	UserError(ctx, w, usererror.ErrInternal)
}

// UserError writes the json-encoded user error.
func UserError(ctx context.Context, w http.ResponseWriter, err *usererror.Error) {
	log.Ctx(ctx).Debug().Err(err).Msgf("operation resulted in user facing error")

	JSON(w, err.Status, err)
}
