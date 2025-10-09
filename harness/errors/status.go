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

package errors

import (
	"errors"
	"fmt"
)

type Status string

const (
	StatusConflict           Status = "conflict"
	StatusInternal           Status = "internal"
	StatusInvalidArgument    Status = "invalid"
	StatusNotFound           Status = "not_found"
	StatusNotImplemented     Status = "not_implemented"
	StatusUnauthorized       Status = "unauthorized"
	StatusForbidden          Status = "forbidden"
	StatusFailed             Status = "failed"
	StatusPreconditionFailed Status = "precondition_failed"
	StatusAborted            Status = "aborted"
)

type Error struct {
	// Machine-readable status code.
	Status Status

	// Human-readable error message.
	Message string

	// Source error
	Err error

	// Details
	Details map[string]any
}

func (e *Error) SetErr(err error) *Error {
	e.Err = err
	return e
}

func (e *Error) SetDetails(details map[string]any) *Error {
	e.Details = details
	return e
}

func (e *Error) Unwrap() error {
	return e.Err
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s", e.Message, e.Err)
	}

	return e.Message
}

// AsStatus unwraps an error and returns its code.
// Non-application errors always return StatusInternal.
func AsStatus(err error) Status {
	if err == nil {
		return ""
	}
	e := AsError(err)
	if e != nil {
		return e.Status
	}
	return StatusInternal
}

// Message unwraps an error and returns its message.
func Message(err error) string {
	if err == nil {
		return ""
	}
	e := AsError(err)
	if e != nil {
		return e.Message
	}
	return err.Error()
}

// Details unwraps an error and returns its details.
func Details(err error) map[string]any {
	if err == nil {
		return nil
	}
	e := AsError(err)
	if e != nil {
		return e.Details
	}
	return nil
}

// AsError return err as Error.
func AsError(err error) (e *Error) {
	if err == nil {
		return nil
	}
	if errors.As(err, &e) {
		return
	}
	return
}

// Format is a helper function to return an Error with a given status and formatted message.
func Format(code Status, format string, args ...interface{}) *Error {
	msg := fmt.Sprintf(format, args...)
	return &Error{
		Status:  code,
		Message: msg,
	}
}

// NotFound is a helper function to return an not found Error.
func NotFound(format string, args ...interface{}) *Error {
	return Format(StatusNotFound, format, args...)
}

// InvalidArgument is a helper function to return an invalid argument Error.
func InvalidArgument(format string, args ...interface{}) *Error {
	return Format(StatusInvalidArgument, format, args...)
}

// Internal is a helper function to return an internal Error.
func Internal(err error, format string, args ...interface{}) *Error {
	msg := fmt.Sprintf(format, args...)
	return Format(StatusInternal, msg).SetErr(
		fmt.Errorf("%s: %w", msg, err),
	)
}

// Conflict is a helper function to return an conflict Error.
func Conflict(format string, args ...interface{}) *Error {
	return Format(StatusConflict, format, args...)
}

// PreconditionFailed is a helper function to return an precondition
// failed error.
func PreconditionFailed(format string, args ...interface{}) *Error {
	return Format(StatusPreconditionFailed, format, args...)
}

// Unauthorized is a helper function to return an unauthorized error.
func Unauthorized(format string, args ...interface{}) *Error {
	return Format(StatusUnauthorized, format, args...)
}

// Forbidden is a helper function to return a forbidden error.
func Forbidden(format string, args ...interface{}) *Error {
	return Format(StatusForbidden, format, args...)
}

// Failed is a helper function to return failed error status.
func Failed(format string, args ...interface{}) *Error {
	return Format(StatusFailed, format, args...)
}

// Aborted is a helper function to return aborted error status.
func Aborted(format string, args ...interface{}) *Error {
	return Format(StatusAborted, format, args...)
}

// IsNotFound checks if err is not found error.
func IsNotFound(err error) bool {
	return AsStatus(err) == StatusNotFound
}

// IsConflict checks if err is conflict error.
func IsConflict(err error) bool {
	return AsStatus(err) == StatusConflict
}

// IsInvalidArgument checks if err is invalid argument error.
func IsInvalidArgument(err error) bool {
	return AsStatus(err) == StatusInvalidArgument
}

// IsInternal checks if err is internal error.
func IsInternal(err error) bool {
	return AsStatus(err) == StatusInternal
}

// IsPreconditionFailed checks if err is precondition failed error.
func IsPreconditionFailed(err error) bool {
	return AsStatus(err) == StatusPreconditionFailed
}

// IsAborted checks if err is aborted error.
func IsAborted(err error) bool {
	return AsStatus(err) == StatusAborted
}
