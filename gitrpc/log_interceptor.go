// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"context"

	"github.com/harness/gitness/gitrpc/rpc"

	"github.com/rs/zerolog/hlog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// ClientLogInterceptor injects the zerlog request ID into the metadata.
// That allows the gitrpc server to log with the same request ID as the client.
type ClientLogInterceptor struct {
}

func NewClientLogInterceptor() ClientLogInterceptor {
	return ClientLogInterceptor{}
}

func (i ClientLogInterceptor) UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx = appendLoggingRequestIDToOutgoingMetadata(ctx)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func (i ClientLogInterceptor) StreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string,
		streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		ctx = appendLoggingRequestIDToOutgoingMetadata(ctx)
		return streamer(ctx, desc, cc, method)
	}
}

// appendLoggingRequestIDToOutgoingMetadata appends the zerolog request ID to the outgoing grpc metadata, if available.
func appendLoggingRequestIDToOutgoingMetadata(ctx context.Context) context.Context {
	if id, ok := hlog.IDFromCtx(ctx); ok {
		ctx = metadata.AppendToOutgoingContext(ctx, rpc.MetadataKeyRequestID, id.String())
	}
	return ctx
}
