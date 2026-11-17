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
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestHLogRequestIDHandlerTraceContext(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tracer := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)).Tracer("test")

	var buf bytes.Buffer
	var spanContext trace.SpanContext

	inner := HLogRequestIDHandler()(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		spanContext = trace.SpanContextFromContext(r.Context())
		zerolog.Ctx(r.Context()).Info().Msg("test message")
	}))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := zerolog.New(&buf).WithContext(r.Context())
		ctx, span := tracer.Start(ctx, r.Method)
		defer span.End()
		inner.ServeHTTP(w, r.WithContext(ctx))
	})

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set(requestIDHeader, "req-123")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Equal(t, "req-123", entry["request_id"])
	assert.Equal(t, spanContext.TraceID().String(), entry["trace_id"])
	assert.Equal(t, spanContext.SpanID().String(), entry["span_id"])

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	assert.Contains(t, spans[0].Attributes(), attribute.String("request_id", "req-123"))
}

func TestHLogRequestIDHandlerWithoutTrace(t *testing.T) {
	var buf bytes.Buffer

	inner := HLogRequestIDHandler()(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		zerolog.Ctx(r.Context()).Info().Msg("test message")
	}))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inner.ServeHTTP(w, r.WithContext(zerolog.New(&buf).WithContext(r.Context())))
	})

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", http.NoBody))

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.NotEmpty(t, entry["request_id"])
	assert.NotContains(t, entry, "trace_id")
	assert.NotContains(t, entry, "span_id")
}
