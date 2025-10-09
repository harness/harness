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

package logging

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Option allows to annotate logs with metadata.
type Option func(c zerolog.Context) zerolog.Context

// UpdateContext updates the existing logging context in the context.
// IMPORTANT: No new context is created, all future logs with provided context will be impacted.
func UpdateContext(ctx context.Context, opts ...Option) {
	// updates existing logging context in provided context.Context
	zerolog.Ctx(ctx).UpdateContext(func(c zerolog.Context) zerolog.Context {
		for _, opt := range opts {
			c = opt(c)
		}

		return c
	})
}

// NewContext derives a new LoggingContext from the existing LoggingContext, adds the provided annotations,
// and then returns a clone of the provided context.Context with the new LoggingContext.
// IMPORTANT: The provided context is not modified, logging annotations are only part of the new context.
func NewContext(ctx context.Context, opts ...Option) context.Context {
	// create child of current context
	childloggingContext := log.Ctx(ctx).With()

	// update child context
	for _, opt := range opts {
		childloggingContext = opt(childloggingContext)
	}

	// return copied context with new logging context
	return childloggingContext.Logger().WithContext(ctx)
}

// WithRequestID can be used to annotate logs with the request id.
func WithRequestID(reqID string) Option {
	return func(c zerolog.Context) zerolog.Context {
		return c.Str("request_id", reqID)
	}
}
