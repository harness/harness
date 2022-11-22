// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package middleware

import (
	"context"
	"errors"
	"reflect"

	"github.com/harness/gitness/gitrpc/internal/types"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ErrInterceptor struct {
}

func NewErrInterceptor() *ErrInterceptor {
	return &ErrInterceptor{}
}

func (i *ErrInterceptor) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (interface{}, error) {
		value, err := handler(ctx, req)
		if (value == nil || reflect.ValueOf(value).IsNil()) && err == nil {
			return nil, status.Error(codes.Internal, "service returned no error and no object")
		}
		err = i.processError(err)
		return value, err
	}
}

func (i *ErrInterceptor) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo,
		handler grpc.StreamHandler) error {
		err := handler(srv, stream)
		err = i.processError(err)
		return err
	}
}

func (i *ErrInterceptor) processError(err error) error {
	if err == nil {
		return nil
	}

	message := err.Error()

	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, message)
	case errors.Is(err, types.ErrNotFound):
		return status.Error(codes.NotFound, message)
	case errors.Is(err, types.ErrAlreadyExists):
		return status.Error(codes.AlreadyExists, message)
	case errors.Is(err, types.ErrInvalidArgument):
		return status.Error(codes.InvalidArgument, message)
	case errors.Is(err, types.ErrInvalidPath):
		return status.Error(codes.InvalidArgument, message)
	case errors.Is(err, types.ErrUndefinedAction):
		return status.Error(codes.InvalidArgument, message)
	case errors.Is(err, types.ErrHeaderCannotBeEmpty):
		return status.Error(codes.InvalidArgument, message)
	case errors.Is(err, types.ErrActionListEmpty):
		return status.Error(codes.FailedPrecondition, message)
	case errors.Is(err, types.ErrContentSentBeforeAction):
		return status.Error(codes.FailedPrecondition, message)
	default:
		return status.Errorf(codes.Unknown, message)
	}
}
