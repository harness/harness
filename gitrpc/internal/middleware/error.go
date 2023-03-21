// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package middleware

import (
	"context"
	"errors"
	"reflect"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ErrInterceptor struct {
}

func NewErrInterceptor() ErrInterceptor {
	return ErrInterceptor{}
}

func (i ErrInterceptor) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (interface{}, error) {
		value, err := handler(ctx, req)
		if (value == nil || reflect.ValueOf(value).IsNil()) && err == nil {
			return nil, status.Error(codes.Internal, "service returned no error and no object")
		}
		err = processError(ctx, err)
		return value, err
	}
}

func (i ErrInterceptor) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo,
		handler grpc.StreamHandler) error {
		err := handler(srv, stream)
		err = processError(stream.Context(), err)
		return err
	}
}

func processError(ctx context.Context, err error) (rerr error) {
	if err == nil {
		return nil
	}

	defer func() {
		statusErr, ok := status.FromError(rerr)
		if !ok {
			return
		}
		//nolint: exhaustive // log only server side errors, no need to log user based errors
		switch statusErr.Code() {
		case codes.Unknown,
			codes.DeadlineExceeded,
			codes.ResourceExhausted,
			codes.FailedPrecondition,
			codes.Aborted,
			codes.OutOfRange,
			codes.Unimplemented,
			codes.Internal,
			codes.Unavailable,
			codes.DataLoss:
			{
				logCtx := log.Ctx(ctx)
				logCtx.Error().Msg(err.Error())
			}
		}
	}()

	// custom errors should implement StatusError
	var statusError interface {
		Status() (*status.Status, error)
	}

	if errors.As(err, &statusError) {
		st, sterr := statusError.Status()
		if sterr != nil {
			return sterr
		}
		return st.Err()
	}

	if status, ok := status.FromError(err); ok {
		return status.Err()
	}

	return status.Errorf(codes.Unknown, err.Error())
}
