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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestTraceContextRoundTrip(t *testing.T) {
	tracer := sdktrace.NewTracerProvider().Tracer("test")
	ctx, span := tracer.Start(context.Background(), "parent")
	defer span.End()

	traceContext := TraceContext(ctx)
	require.Contains(t, traceContext, "traceparent")

	header := http.Header{}
	SetHTTPHeader(header, traceContext)

	want := http.Header{}
	InjectHTTPHeader(ctx, want)
	assert.Equal(t, want, header)
}

func TestTraceContextWithoutSpan(t *testing.T) {
	assert.Empty(t, TraceContext(context.Background()))
}

func TestMapRoundTrip(t *testing.T) {
	tracer := sdktrace.NewTracerProvider().Tracer("test")
	ctx, span := tracer.Start(context.Background(), "parent")
	defer span.End()

	values := map[string]any{"event": []byte("payload")}
	InjectMap(ctx, values)
	traceparent, ok := values["traceparent"].(string)
	require.True(t, ok)
	assert.Equal(t, []byte("payload"), values["event"])

	// Redis returns strings, the in-memory stream keeps the original values.
	for name, value := range map[string]any{
		"string": traceparent,
		"bytes":  []byte(traceparent),
	} {
		t.Run(name, func(t *testing.T) {
			sc := trace.SpanContextFromContext(ContextFromMap(context.Background(), map[string]any{"traceparent": value}))
			assert.Equal(t, span.SpanContext().TraceID(), sc.TraceID())
			assert.Equal(t, span.SpanContext().SpanID(), sc.SpanID())
			assert.True(t, sc.IsRemote())
		})
	}
}

func TestContextFromMapWithoutTraceContext(t *testing.T) {
	ctx := ContextFromMap(context.Background(), map[string]any{"event": []byte("payload")})
	assert.False(t, trace.SpanContextFromContext(ctx).IsValid())
}
