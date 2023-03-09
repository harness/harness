// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package render

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

// indent the json-encoded API responses.
var indent bool

func init() {
	indent, _ = strconv.ParseBool(
		os.Getenv("HTTP_JSON_INDENT"),
	)
}

// TranslatedUserError writes the translated user error of the provided error.
func TranslatedUserError(w http.ResponseWriter, err error) {
	log.Warn().Msgf("operation resulted in user facing error. Internal details: %s", err)
	UserError(w, usererror.Translate(err))
}

// NotFound writes the json-encoded message for a not found error.
func NotFound(w http.ResponseWriter) {
	UserError(w, usererror.ErrNotFound)
}

// Unauthorized writes the json-encoded message for an unauthorized error.
func Unauthorized(w http.ResponseWriter) {
	UserError(w, usererror.ErrUnauthorized)
}

// Forbidden writes the json-encoded message for a forbidden error.
func Forbidden(w http.ResponseWriter) {
	UserError(w, usererror.ErrForbidden)
}

// BadRequest writes the json-encoded message for a bad request error.
func BadRequest(w http.ResponseWriter) {
	UserError(w, usererror.ErrBadRequest)
}

// BadRequestError writes the json-encoded error with a bad request status code.
func BadRequestError(w http.ResponseWriter, err *usererror.Error) {
	UserError(w, err)
}

// BadRequest writes the json-encoded message with a bad request status code.
func BadRequestf(w http.ResponseWriter, format string, args ...interface{}) {
	ErrorMessagef(w, http.StatusBadRequest, format, args...)
}

// InternalError writes the json-encoded message for an internal error.
func InternalError(w http.ResponseWriter) {
	UserError(w, usererror.ErrInternal)
}

// ErrorMessagef writes the json-encoded, formated error message.
func ErrorMessagef(w http.ResponseWriter, code int, format string, args ...interface{}) {
	JSON(w, code, &usererror.Error{Message: fmt.Sprintf(format, args...)})
}

// UserError writes the json-encoded user error.
func UserError(w http.ResponseWriter, err *usererror.Error) {
	JSON(w, err.Status, err)
}

// DeleteSuccessful writes the header for a successful delete.
func DeleteSuccessful(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// JSON writes the json-encoded value to the response
// with the provides status.
func JSON(w http.ResponseWriter, code int, v interface{}) {
	// set common headers
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// flush the headers - before body or status will be 200 OK
	w.WriteHeader(code)

	// write body
	enc := json.NewEncoder(w)
	if indent { // is this necessary? it will affect performance
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(v); err != nil {
		log.Err(err).Msgf("Failed to write json encoding to response body.")
	}
}

// JSONArrayDynamic outputs an JSON array whose elements are streamed from a channel.
// Due to the dynamic nature (unknown number of elements) the function will use
// chunked transfer encoding for large files.
func JSONArrayDynamic[T comparable](ctx context.Context, w http.ResponseWriter, stream types.Stream[T]) {
	count := 0
	enc := json.NewEncoder(w)

	for {
		data, err := stream.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			// User canceled the request - no need to do anything
			if errors.Is(err, context.Canceled) {
				return
			}

			if count == 0 {
				// Write the error only if no data has been streamed yet.
				TranslatedUserError(w, err)
				return
			}

			// Array data has been already streamed, it's too late for the output - so just log and quit.
			log.Ctx(ctx).Warn().Msgf("Failed to write JSON array response body: %v", err)
			return
		}

		if count == 0 {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte{'['})
		} else {
			_, _ = w.Write([]byte{','})
		}

		count++

		_ = enc.Encode(data)
	}

	if count == 0 {
		_, _ = w.Write([]byte{'['})
	}

	_, _ = w.Write([]byte{']'})
}
