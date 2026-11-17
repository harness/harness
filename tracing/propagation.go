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

package tracing

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel/propagation"
)

// propagator is used explicitly, also by short-lived processes (e.g. git hooks) that never call Init.
var propagator = propagation.NewCompositeTextMapPropagator(
	propagation.TraceContext{},
	propagation.Baggage{},
)

// TraceContext returns the trace context of ctx (traceparent, tracestate, baggage) as a map, to pass it to
// another process as part of its input (e.g. the githook payload). It's empty if ctx carries no trace context.
func TraceContext(ctx context.Context) map[string]string {
	carrier := propagation.MapCarrier{}
	propagator.Inject(ctx, carrier)

	return carrier
}

// SetHTTPHeader writes a trace context returned by TraceContext to the provided HTTP headers.
func SetHTTPHeader(header http.Header, traceContext map[string]string) {
	for key, value := range traceContext {
		header.Set(key, value)
	}
}

// InjectHTTPHeader writes the trace context of ctx to the provided HTTP headers.
func InjectHTTPHeader(ctx context.Context, header http.Header) {
	propagator.Inject(ctx, propagation.HeaderCarrier(header))
}

// InjectMap writes the trace context of ctx to the provided map (e.g. the fields of a stream message).
func InjectMap(ctx context.Context, values map[string]any) {
	carrier := propagation.MapCarrier{}
	propagator.Inject(ctx, carrier)

	for key, value := range carrier {
		values[key] = value
	}
}

// ContextFromMap returns a copy of ctx with the trace context read from a map written by InjectMap.
// Values can be strings or byte slices, as stream messages read from Redis contain strings.
func ContextFromMap(ctx context.Context, values map[string]any) context.Context {
	carrier := propagation.MapCarrier{}
	for _, key := range propagator.Fields() {
		switch value := values[key].(type) {
		case string:
			carrier.Set(key, value)
		case []byte:
			carrier.Set(key, string(value))
		}
	}

	return propagator.Extract(ctx, carrier)
}
