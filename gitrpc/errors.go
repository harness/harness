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

package gitrpc

import (
	"errors"
	"fmt"

	"github.com/harness/gitness/gitrpc/rpc"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrNoParamsProvided = ErrInvalidArgumentf("params not provided")
)

const (
	conflictFilesKey = "conflict_files"
	pathKey          = "path"
)

type Status string

const (
	StatusConflict           Status = "conflict"
	StatusInternal           Status = "internal"
	StatusInvalidArgument    Status = "invalid"
	StatusNotFound           Status = "not_found"
	StatusPathNotFound       Status = "path_not_found"
	StatusNotImplemented     Status = "not_implemented"
	StatusUnauthorized       Status = "unauthorized"
	StatusFailed             Status = "failed"
	StatusPreconditionFailed Status = "precondition_failed"
	StatusNotMergeable       Status = "not_mergeable"
	StatusAborted            Status = "aborted"
)

type Error struct {
	// Machine-readable status code.
	Status Status

	// Human-readable error message.
	Message string

	// Details
	Details map[string]any
}

// Error implements the error interface.
func (e *Error) Error() string {
	return e.Message
}

// ErrorStatus unwraps an gitrpc error and returns its code.
// Non-application errors always return StatusInternal.
func ErrorStatus(err error) Status {
	var (
		e *Error
	)
	if err == nil {
		return ""
	}
	if errors.As(err, &e) {
		return e.Status
	}
	return StatusInternal
}

// ErrorMessage unwraps an gitrpc error and returns its message.
// Non-gitrpc errors always return "Internal error".
func ErrorMessage(err error) string {
	var (
		e *Error
	)
	if err == nil {
		return ""
	}
	if errors.As(err, &e) {
		return e.Message
	}
	return "Internal error."
}

// ErrorDetails unwraps an gitrpc error and returns its details.
// Non-gitrpc errors always return nil.
func ErrorDetails(err error) map[string]any {
	var (
		e *Error
	)
	if err == nil {
		return nil
	}
	if errors.As(err, &e) {
		return e.Details
	}
	return nil
}

// NewError is a factory function to return an Error with a given status and message.
func NewError(code Status, message string) *Error {
	return &Error{
		Status:  code,
		Message: message,
	}
}

// NewError is a factory function to return an Error with a given status, message and details.
func NewErrorWithDetails(code Status, message string, details map[string]any) *Error {
	err := NewError(code, message)
	err.Details = details
	return err
}

// Errorf is a helper function to return an Error with a given status and formatted message.
func Errorf(code Status, format string, args ...interface{}) *Error {
	return &Error{
		Status:  code,
		Message: fmt.Sprintf(format, args...),
	}
}

// ErrInvalidArgumentf is a helper function to return an invalid argument Error.
func ErrInvalidArgumentf(format string, args ...interface{}) *Error {
	return Errorf(StatusInvalidArgument, format, args...)
}

func processRPCErrorf(err error, format string, args ...interface{}) error {
	if errors.Is(err, &Error{}) {
		return err
	}
	// create fallback error returned if we can't map it
	fallbackMsg := fmt.Sprintf(format, args...)
	fallbackErr := NewError(StatusInternal, fallbackMsg)

	// ensure it's an rpc error
	st, ok := status.FromError(err)
	if !ok {
		return fallbackErr
	}

	msg := st.Message()

	switch {
	case st.Code() == codes.AlreadyExists:
		return NewError(StatusConflict, msg)
	case st.Code() == codes.NotFound:
		code := StatusNotFound
		details := make(map[string]any)
		for _, detail := range st.Details() {
			switch t := detail.(type) {
			case *rpc.PathNotFoundError:
				code = StatusPathNotFound
				details[pathKey] = t.Path
			default:
			}
		}
		if len(details) > 0 {
			return NewErrorWithDetails(code, msg, details)
		}
		return NewError(code, msg)
	case st.Code() == codes.InvalidArgument:
		return NewError(StatusInvalidArgument, msg)
	case st.Code() == codes.FailedPrecondition:
		code := StatusPreconditionFailed
		details := make(map[string]any)
		for _, detail := range st.Details() {
			switch t := detail.(type) {
			case *rpc.MergeConflictError:
				details[conflictFilesKey] = t.ConflictingFiles
				code = StatusNotMergeable
			default:
			}
		}
		if len(details) > 0 {
			return NewErrorWithDetails(code, msg, details)
		}
		return NewError(code, msg)
	default:
		return fallbackErr
	}
}

func AsConflictFilesError(err error) (files []string) {
	details := ErrorDetails(err)
	object, ok := details[conflictFilesKey]
	if ok {
		files, _ = object.([]string)
	}

	return
}

// AsPathNotFoundError returns the path that wasn't found in case that's the error.
func AsPathNotFoundError(err error) (path string) {
	details := ErrorDetails(err)
	object, ok := details[pathKey]
	if ok {
		path, _ = object.(string)
	}

	return
}
