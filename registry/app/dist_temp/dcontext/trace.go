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
	"runtime"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// WithTrace allocates a traced timing span in a new context. This allows a
// caller to track the time between calling WithTrace and the returned done
// function. When the done function is called, a log message is emitted with a
// "trace.duration" field, corresponding to the elapsed time and a
// "trace.func" field, corresponding to the function that called WithTrace.
//
// The logging keys "trace.id" and "trace.parent.id" are provided to implement
// dapper-like tracing. This function should be complemented with a WithSpan
// method that could be used for tracing distributed RPC calls.
//
// The main benefit of this function is to post-process log messages or
// intercept them in a hook to provide timing data. Trace ids and parent ids
// can also be linked to provide call tracing, if so required.
//
// Here is an example of the usage:
//
//	func timedOperation(ctx Context) {
//		ctx, done := WithTrace(ctx)
//		defer done("this will be the log message")
//		// ... function body ...
//	}
//
// If the function ran for roughly 1s, such a usage would emit a log message
// as follows:
//
//	INFO[0001] this will be the log message  trace.duration=1.004575763s
//
// trace.func=github.com/distribution/distribution/context.traceOperation trace.id=<id> ...
//
// Notice that the function name is automatically resolved, along with the
// package and a trace id is emitted that can be linked with parent ids.
func WithTrace(ctx context.Context) (context.Context, func(format string, a ...interface{})) {
	if ctx == nil {
		ctx = Background()
	}

	pc, file, line, _ := runtime.Caller(1)
	f := runtime.FuncForPC(pc)
	ctx = &traced{
		Context: ctx,
		id:      uuid.NewString(),
		start:   time.Now(),
		parent:  GetStringValue(ctx, "trace.id"),
		fnname:  f.Name(),
		file:    file,
		line:    line,
	}

	return ctx, func(format string, a ...interface{}) {
		GetLogger(
			ctx, log.Info(),
			"trace.duration",
			"trace.id",
			"trace.parent.id",
			"trace.func",
			"trace.file",
			"trace.line",
		).
			Msgf(format, a...)
	}
}

// traced represents a context that is traced for function call timing. It
// also provides fast lookup for the various attributes that are available on
// the trace.
type traced struct {
	context.Context
	id     string
	parent string
	start  time.Time
	fnname string
	file   string
	line   int
}

func (ts *traced) Value(key interface{}) interface{} {
	switch key {
	case "trace.start":
		return ts.start
	case "trace.duration":
		return time.Since(ts.start)
	case "trace.id":
		return ts.id
	case "trace.parent.id":
		if ts.parent == "" {
			return nil // must return nil to signal no parent.
		}

		return ts.parent
	case "trace.func":
		return ts.fnname
	case "trace.file":
		return ts.file
	case "trace.line":
		return ts.line
	}

	return ts.Context.Value(key)
}
