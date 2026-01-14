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

package git

import "context"

const (
	RequestIDNone string = "git_none"
)

// requestIDKey is context key for storing and retrieving the request ID to and from a context.
type requestIDKey struct{}

// RequestIDFrom retrieves the request id from the context.
// If no request id exists, RequestIDNone is returned.
func RequestIDFrom(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey{}).(string); ok {
		return v
	}

	return RequestIDNone
}

// WithRequestID returns a copy of parent in which the request id value is set.
// This can be used by external entities to pass request IDs.
func WithRequestID(parent context.Context, v string) context.Context {
	return context.WithValue(parent, requestIDKey{}, v)
}
