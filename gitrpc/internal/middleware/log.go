// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package middleware

import (
	"context"
	"strings"
	"time"

	"github.com/harness/gitness/gitrpc/rpc"

	"github.com/rs/xid"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const (
	RequestIDNone string = "gitrpc_none"
)

// requestIDKey is context key for storing and retrieving the request ID to and from a context.
type requestIDKey struct{}

// LogInterceptor injects a zerolog logger with common grpc related annotations and logs the completion of the call.
// If the metadata contains a request id, the logger is annotated with the same request ID, otherwise with a new one.
type LogInterceptor struct {
}

func NewLogInterceptor() LogInterceptor {
	return LogInterceptor{}
}

func (i LogInterceptor) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (interface{}, error) {
		ctx = injectLogging(ctx, info.FullMethod)

		// measure execution time
		start := time.Now()
		value, err := handler(ctx, req)

		logCompletion(ctx, start, err)

		return value, err
	}
}

func (i LogInterceptor) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo,
		handler grpc.StreamHandler) error {
		ctx := injectLogging(stream.Context(), info.FullMethod)

		// wrap stream with updated context
		stream = &logServerStream{
			ServerStream: stream,
			ctx:          ctx,
		}

		// measure execution time
		start := time.Now()
		err := handler(srv, stream)

		logCompletion(ctx, start, err)

		return err
	}
}

// WithRequestID returns a copy of parent in which the request id value is set.
func WithRequestID(parent context.Context, v string) context.Context {
	return context.WithValue(parent, requestIDKey{}, v)
}

// RequestIDFrom retrieves the request id from the context.
// If no request id exists, RequestIDNone is returned.
func RequestIDFrom(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey{}).(string); ok {
		return v
	}

	return RequestIDNone
}

func injectLogging(ctx context.Context, fullMethod string) context.Context {
	// split fullMethod into service and method (expected format: "/package.service/method...")
	// If it doesn't match the expected format, the full string is put into method.
	service, method := "", fullMethod
	if len(fullMethod) > 0 && fullMethod[0] == '/' {
		if s, m, ok := strings.Cut(fullMethod[1:], "/"); ok {
			service, method = s, m
		}
	}

	// get request id (or create a new one) and inject it for later usage (git env variables)
	requestID := getOrCreateRequestID(ctx)
	ctx = WithRequestID(ctx, requestID)

	// create new logCtx with injected info
	logCtx := log.Logger.With().
		Str("grpc.service", service).
		Str("grpc.method", method).
		Str("grpc.request_id", requestID)

	// add peer information if available
	if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
		logCtx = logCtx.Str("grpc.peer", p.Addr.String())
	}

	// inject logger in context
	logger := logCtx.Logger()
	return logger.WithContext(ctx)
}

func logCompletion(ctx context.Context, start time.Time, err error) {
	logCtx := log.Ctx(ctx).Info().
		Dur("grpc.elapsed_ms", time.Since(start))

	// try to get grpc status code
	if status, ok := status.FromError(err); ok {
		logCtx.Str("grpc.status_code", status.Code().String())
	}

	logCtx.Msg("grpc request completed.")
}

func getOrCreateRequestID(ctx context.Context) string {
	// check if request id was passed as part of grpc metadata
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if ids := md.Get(rpc.MetadataKeyRequestID); len(ids) > 0 {
			return ids[0]
		}
	}

	// use same type of request IDs as used by zerolog
	return xid.New().String()
}

// logServerStream is used to modify the stream context.
// In order to modify the stream context we have to create a new struct and overshadow the `Context()` method.
type logServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *logServerStream) Context() context.Context {
	return s.ctx
}
