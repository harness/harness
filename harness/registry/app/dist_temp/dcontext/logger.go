// Source: https://github.com/distribution/distribution

// Copyright 2014 https://github.com/distribution/distribution Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dcontext

import (
	"context"
	"fmt"
	"runtime"

	"github.com/rs/zerolog"
)

// Logger provides a leveled-logging interface.
type Logger interface {
	Msgf(format string, v ...interface{})

	Msg(msg string)
}

type loggerKey struct{}

// WithLogger creates a new context with provided logger.
func WithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// GetLoggerWithFields returns a logger instance with the specified fields
// without affecting the context. Extra specified keys will be resolved from
// the context.
func GetLoggerWithFields(
	ctx context.Context, log *zerolog.Event,
	fields map[interface{}]interface{}, keys ...interface{},
) Logger {
	logger := getZerologLogger(ctx, log, keys...)
	for key, value := range fields {
		logger.Interface(fmt.Sprint(key), value)
	}

	return logger
}

// GetLogger returns the logger from the current context, if present. If one
// or more keys are provided, they will be resolved on the context and
// included in the logger. While context.Value takes an interface, any key
// argument passed to GetLogger will be passed to fmt.Sprint when expanded as
// a logging key field. If context keys are integer constants, for example,
// its recommended that a String method is implemented.
func GetLogger(ctx context.Context, l *zerolog.Event, keys ...interface{}) Logger {
	return getZerologLogger(ctx, l, keys...)
}

// getZerologLogger returns the zerolog logger for the context. If one more keys
// are provided, they will be resolved on the context and included in the
// logger. Only use this function if specific zerolog functionality is
// required.
func getZerologLogger(ctx context.Context, l *zerolog.Event, keys ...interface{}) *zerolog.Event {
	var logger *zerolog.Event

	// Get a logger, if it is present.
	loggerInterface := ctx.Value(loggerKey{})
	if loggerInterface != nil {
		if lgr, ok := loggerInterface.(*zerolog.Event); ok {
			logger = lgr
		}
	}

	if logger == nil {
		logger = l.Str("go.version", runtime.Version())
		// Fill in the instance id, if we have it.
		instanceID := ctx.Value("instance.id")
		if instanceID != nil {
			logger.Interface("instance.id", instanceID)
		}
	}

	for _, key := range keys {
		v := ctx.Value(key)
		if v != nil {
			logger.Interface(fmt.Sprint(key), v)
		}
	}

	return logger
}
