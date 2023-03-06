// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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

type Status string

const (
	StatusConflict           Status = "conflict"
	StatusInternal           Status = "internal"
	StatusInvalidArgument    Status = "invalid"
	StatusNotFound           Status = "not_found"
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
	// create fallback error returned if we can't map it
	fallbackErr := fmt.Errorf(format, args...)

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
		return NewError(StatusNotFound, msg)
	case st.Code() == codes.InvalidArgument:
		return NewError(StatusInvalidArgument, msg)
	case st.Code() == codes.FailedPrecondition:
		code := StatusPreconditionFailed
		details := make(map[string]any)
		for _, detail := range st.Details() {
			switch t := detail.(type) {
			case *rpc.MergeConflictError:
				details["conflict_files"] = t.ConflictingFiles
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
