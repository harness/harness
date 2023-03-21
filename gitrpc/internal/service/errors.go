// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/harness/gitness/gitrpc/internal/types"
	"github.com/harness/gitness/gitrpc/rpc"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

type Error struct {
	Code    codes.Code
	Message string
	Err     error
	details []proto.Message
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s, err: %v", e.Message, e.Err.Error())
}

func (e *Error) Status() (*status.Status, error) {
	st := status.New(e.Code, e.Message)
	if len(e.details) == 0 {
		return st, nil
	}
	// add details
	proto := st.Proto()
	for _, detail := range e.details {
		marshaled, err := anypb.New(detail)
		if err != nil {
			return nil, err
		}

		proto.Details = append(proto.Details, marshaled)
	}
	return status.FromProto(proto), nil
}

func (e *Error) Details() any {
	return e.details
}

func (e *Error) Unwrap() error {
	return e.Err
}

// Errorf generates new Error with status code and custom arguments.
// args can contain format args and additional arg like err which will be logged
// by middleware and details object type of map. Ordering of args element
// should first process format args and then error or detail.
func Errorf(code codes.Code, format string, args ...any) (err error) {
	details := make([]proto.Message, 0, 8)
	newargs := make([]any, 0, len(args))

	for _, arg := range args {
		switch t := arg.(type) {
		case error:
			err = t
		case proto.Message:
			details = append(details, t)
		default:
			newargs = append(newargs, arg)
		}
	}

	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, newargs...),
		Err:     err,
		details: details,
	}
}

func wrapError(code codes.Code, err error) error {
	var e *Error
	if errors.As(err, &e) {
		return err
	}
	return &Error{
		Code:    code,
		Message: err.Error(),
		Err:     err,
	}
}

// ErrCanceled wraps err with codes.Canceled, unless err is already a Error error.
func ErrCanceled(err error) error { return wrapError(codes.Canceled, err) }

// ErrDeadlineExceeded wraps err with codes.DeadlineExceeded, unless err is already a Error error.
func ErrDeadlineExceeded(err error) error { return wrapError(codes.DeadlineExceeded, err) }

// ErrInternal wraps err with codes.Internal, unless err is already a Error error.
func ErrInternal(err error) error { return wrapError(codes.Internal, err) }

// ErrInvalidArgument wraps err with codes.InvalidArgument, unless err is already a Error error.
func ErrInvalidArgument(err error) error { return wrapError(codes.InvalidArgument, err) }

// ErrNotFound wraps error with codes.NotFound, unless err is already a Error error.
func ErrNotFound(err error) error { return wrapError(codes.NotFound, err) }

// ErrFailedPrecondition wraps err with codes.FailedPrecondition, unless err is already a Error
// error.
func ErrFailedPrecondition(err error) error { return wrapError(codes.FailedPrecondition, err) }

// ErrUnavailable wraps err with codes.Unavailable, unless err is already a gRPC error.
func ErrUnavailable(err error) error { return wrapError(codes.Unavailable, err) }

// ErrPermissionDenied wraps err with codes.PermissionDenied, unless err is already a Error error.
func ErrPermissionDenied(err error) error { return wrapError(codes.PermissionDenied, err) }

// ErrAlreadyExists wraps err with codes.AlreadyExists, unless err is already a Error error.
func ErrAlreadyExists(err error) error { return wrapError(codes.AlreadyExists, err) }

// ErrAborted wraps err with codes.Aborted, unless err is already a Error type.
func ErrAborted(err error) error { return wrapError(codes.Aborted, err) }

// ErrCanceledf wraps a formatted error with codes.Canceled, unless the formatted error is a
// wrapped Error error.
func ErrCanceledf(format string, a ...interface{}) error {
	return Errorf(codes.Canceled, format, a...)
}

// ErrDeadlineExceededf wraps a formatted error with codes.DeadlineExceeded, unless the formatted
// error is a wrapped Error error.
func ErrDeadlineExceededf(format string, a ...interface{}) error {
	return Errorf(codes.DeadlineExceeded, format, a...)
}

// ErrInternalf wraps a formatted error with codes.Internal, unless the formatted error is a
// wrapped Error error.
func ErrInternalf(format string, a ...interface{}) error {
	return Errorf(codes.Internal, format, a...)
}

// ErrInvalidArgumentf wraps a formatted error with codes.InvalidArgument, unless the formatted
// error is a wrapped Error error.
func ErrInvalidArgumentf(format string, a ...interface{}) error {
	return Errorf(codes.InvalidArgument, format, a...)
}

// ErrNotFoundf wraps a formatted error with codes.NotFound, unless the
// formatted error is a wrapped Error error.
func ErrNotFoundf(format string, a ...interface{}) error {
	return Errorf(codes.NotFound, format, a...)
}

// ErrFailedPreconditionf wraps a formatted error with codes.FailedPrecondition, unless the
// formatted error is a wrapped Error error.
func ErrFailedPreconditionf(format string, a ...interface{}) error {
	return Errorf(codes.FailedPrecondition, format, a...)
}

// ErrUnavailablef wraps a formatted error with codes.Unavailable, unless the
// formatted error is a wrapped Error error.
func ErrUnavailablef(format string, a ...interface{}) error {
	return Errorf(codes.Unavailable, format, a...)
}

// ErrPermissionDeniedf wraps a formatted error with codes.PermissionDenied, unless the formatted
// error is a wrapped Error error.
func ErrPermissionDeniedf(format string, a ...interface{}) error {
	return Errorf(codes.PermissionDenied, format, a...)
}

// ErrAlreadyExistsf wraps a formatted error with codes.AlreadyExists, unless the formatted error is
// a wrapped Error error.
func ErrAlreadyExistsf(format string, a ...interface{}) error {
	return Errorf(codes.AlreadyExists, format, a...)
}

// ErrAbortedf wraps a formatted error with codes.Aborted, unless the formatted error is a wrapped
// Error error.
func ErrAbortedf(format string, a ...interface{}) error {
	return Errorf(codes.Aborted, format, a...)
}

// processGitErrorf translates error.
func processGitErrorf(err error, format string, args ...interface{}) error {
	var (
		cferr *types.MergeConflictsError
	)
	const nl = "\n"
	// when we add err as argument it will be part of the new error
	args = append(args, err)
	switch {
	case errors.Is(err, types.ErrNotFound):
		return ErrNotFoundf(format, args...)
	case errors.Is(err, types.ErrAlreadyExists):
		return ErrAlreadyExistsf(format, args...)
	case errors.Is(err, types.ErrInvalidArgument):
		return ErrInvalidArgumentf(format, args...)
	case errors.As(err, &cferr):
		stdout := strings.Trim(cferr.StdOut, nl)
		conflictingFiles := strings.Split(stdout, nl)
		files := &rpc.MergeConflictError{
			ConflictingFiles: conflictingFiles,
		}
		return ErrFailedPreconditionf("merging failed! conflict files error", files, err)
	case types.IsMergeUnrelatedHistoriesError(err):
		return ErrFailedPreconditionf(format, args...)
	default:
		return Errorf(codes.Unknown, format, args...)
	}
}
