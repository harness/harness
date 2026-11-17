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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

func TestHTTPRoute(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tracer := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)).Tracer("test")

	r := chi.NewRouter()
	r.Use(HTTPRoute())
	r.Get("/repos/{repo_ref}/commits", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx, span := tracer.Start(req.Context(), req.Method)
		defer span.End()
		r.ServeHTTP(w, req.WithContext(ctx))
	})
	req := httptest.NewRequest(http.MethodGet, "/repos/space%2Frepo/commits", http.NoBody)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	assert.Equal(t, "GET /repos/{repo_ref}/commits", spans[0].Name())
	assert.Contains(t, spans[0].Attributes(), semconv.HTTPRoute("/repos/{repo_ref}/commits"))
}

func TestHTTPTransport(t *testing.T) {
	resetGlobals(t)
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	otel.SetTracerProvider(tp)
	tracer := tp.Tracer("test")

	var gotTraceparent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTraceparent = r.Header.Get("traceparent")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, parent := tracer.Start(t.Context(), "parent")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/accounts/abc", http.NoBody)
	require.NoError(t, err)

	client := &http.Client{Transport: HTTPTransport(nil)}
	resp, err := client.Do(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	parent.End()

	spans := recorder.Ended()
	require.Len(t, spans, 2)
	clientSpan := spans[0]
	assert.Equal(t, "GET "+req.URL.Host, clientSpan.Name())
	assert.Equal(t, parent.SpanContext().TraceID(), clientSpan.SpanContext().TraceID())
	assert.Contains(t, gotTraceparent, clientSpan.SpanContext().TraceID().String())
}

func TestExternalHTTPTransport(t *testing.T) {
	resetGlobals(t)
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	otel.SetTracerProvider(tp)

	var gotTraceparent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTraceparent = r.Header.Get("traceparent")
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	ctx, parent := tp.Tracer("test").Start(t.Context(), "parent")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL+"/services/T000/B000/secret-token", http.NoBody)
	require.NoError(t, err)

	client := &http.Client{Transport: ExternalHTTPTransport(nil)}
	resp, err := client.Do(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	parent.End()

	assert.Empty(t, gotTraceparent, "trace context must not be sent to external systems")

	spans := recorder.Ended()
	require.Len(t, spans, 2)
	clientSpan := spans[0]
	assert.Equal(t, "POST "+req.URL.Host, clientSpan.Name())
	assert.Equal(t, parent.SpanContext().SpanID(), clientSpan.Parent().SpanID())
	assert.Contains(t, clientSpan.Attributes(), semconv.HTTPResponseStatusCode(http.StatusBadGateway))
	assert.Equal(t, codes.Error, clientSpan.Status().Code)
	for _, attr := range clientSpan.Attributes() {
		assert.NotContains(t, attr.Value.String(), "secret-token", "the URL must not be recorded")
	}
}

func TestHTTPRouteWithoutSpan(t *testing.T) {
	r := chi.NewRouter()
	r.Use(HTTPRoute())
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", http.NoBody))

	assert.Equal(t, http.StatusOK, rec.Code)
}
