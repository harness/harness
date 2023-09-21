// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gitrpc

import (
	"context"

	"github.com/harness/gitness/gitrpc/rpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type requestIDKey struct{}

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

// WithRequestID returns a copy of parent in which the request id value is set.
// This can be used by external entities to pass request IDs to gitrpc.
func WithRequestID(parent context.Context, v string) context.Context {
	return context.WithValue(parent, requestIDKey{}, v)
}

// RequestIDFrom returns the value of the request ID key on the
// context - ok is true iff a non-empty value existed.
func RequestIDFrom(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(requestIDKey{}).(string)
	return v, ok && v != ""
}

// appendLoggingRequestIDToOutgoingMetadata appends the zerolog request ID to the outgoing grpc metadata, if available.
func appendLoggingRequestIDToOutgoingMetadata(ctx context.Context) context.Context {
	if id, ok := RequestIDFrom(ctx); ok {
		ctx = metadata.AppendToOutgoingContext(ctx, rpc.MetadataKeyRequestID, id)
	}
	return ctx
}
