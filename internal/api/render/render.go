// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package render

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types/check"
)

// indent the json-encoded API responses
var indent bool

func init() {
	indent, _ = strconv.ParseBool(
		os.Getenv("HTTP_JSON_INDENT"),
	)
}

/*
 * UserfiedErrorOrInternal renders the appropriate user facing message for the provided error.
 * If the error is unknown, an internal error is rendered.
 */
func UserfiedErrorOrInternal(w http.ResponseWriter, err error) {

	if errors.Is(err, check.ErrAny) {
		ErrorObject(w, http.StatusBadRequest, &Error{err.Error()})
	} else if errors.Is(err, store.ErrResourceNotFound) {
		ErrorObject(w, http.StatusNotFound, ErrNotFound)
	} else if errors.Is(err, store.ErrDuplicate) {
		ErrorObject(w, http.StatusBadRequest, ErrDuplicate)
	} else if errors.Is(err, store.ErrPrimaryPathCantBeDeleted) {
		ErrorObject(w, http.StatusBadRequest, ErrPrimaryPathCantBeDeleted)
	} else if errors.Is(err, store.ErrPathTooLong) {
		ErrorObject(w, http.StatusBadRequest, ErrPathTooLong)
	} else if errors.Is(err, store.ErrNoChangeInRequestedMove) {
		ErrorObject(w, http.StatusBadRequest, ErrNoChange)
	} else if errors.Is(err, store.ErrIllegalMoveCyclicHierarchy) {
		ErrorObject(w, http.StatusBadRequest, ErrCyclicHierarchy)
	} else if errors.Is(err, store.ErrSpaceWithChildsCantBeDeleted) {
		ErrorObject(w, http.StatusBadRequest, ErrSpaceWithChildsCantBeDeleted)
	} else {
		// nothing found - render internal error
		fmt.Println(err)
		InternalError(w)
	}
}

// NotFound writes the json-encoded message for a not found error.
func NotFound(w http.ResponseWriter) {
	ErrorObject(w, http.StatusNotFound, ErrNotFound)
}

// Unauthorized writes the json-encoded message for an unauthorized error.
func Unauthorized(w http.ResponseWriter) {
	ErrorObject(w, http.StatusUnauthorized, ErrUnauthorized)
}

// Forbidden writes the json-encoded message for a forbidden error.
func Forbidden(w http.ResponseWriter) {
	ErrorObject(w, http.StatusForbidden, ErrForbidden)
}

// BadRequest writes the json-encoded message for a bad request error.
func BadRequest(w http.ResponseWriter) {
	ErrorObject(w, http.StatusBadRequest, ErrBadRequest)
}

// BadRequestError writes the json-encoded error with a bad request status code.
func BadRequestError(w http.ResponseWriter, err *Error) {
	ErrorObject(w, http.StatusBadRequest, err)
}

// BadRequest writes the json-encoded message with a bad request status code.
func BadRequestf(w http.ResponseWriter, format string, args ...interface{}) {
	ErrorMessagef(w, http.StatusBadRequest, format, args...)
}

// InternalError writes the json-encoded message for an internal error.
func InternalError(w http.ResponseWriter) {
	ErrorObject(w, http.StatusInternalServerError, ErrInternal)
}

// ErrorMessagef writes the json-encoded, formated error message.
func ErrorMessagef(w http.ResponseWriter, code int, format string, args ...interface{}) {
	JSON(w, code, &Error{Message: fmt.Sprintf(format, args...)})
}

// ErrorMessagef writes the json-encoded, formated error message.
func ErrorObject(w http.ResponseWriter, code int, err *Error) {
	JSON(w, code, err)
}

// JSON writes the json-encoded value to the response
// with the provides status
func JSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	enc := json.NewEncoder(w)
	if indent { // is this necessary? it will affect performance
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(v); err != nil {
		code = http.StatusInternalServerError
	}
	w.WriteHeader(code)
}
