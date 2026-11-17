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

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"
)

const scopeName = "github.com/harness/gitness/tracing"

// HTTPHandler wraps the handler to start a server span for every request.
// Spans are named after the HTTP method until HTTPRoute renames them with the matched route.
func HTTPHandler(handler http.Handler) http.Handler {
	return otelhttp.NewHandler(handler, "http.server",
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return r.Method
		}),
	)
}

// HTTPTransport wraps the round tripper to start a client span for every outgoing request and to propagate
// the trace context to the called service. A nil round tripper defaults to http.DefaultTransport.
// Spans are named "<method> <host>" to keep them low-cardinality while still identifying the called service.
func HTTPTransport(rt http.RoundTripper) http.RoundTripper {
	return otelhttp.NewTransport(rt,
		otelhttp.WithPropagators(propagator),
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return r.Method + " " + r.URL.Host
		}),
	)
}

// ExternalHTTPTransport wraps the round tripper to start a client span for every request to a system outside of
// Harness (e.g. webhook endpoints or SCM providers). A nil round tripper defaults to http.DefaultTransport.
// Unlike HTTPTransport, the trace context isn't sent to the called system and only the method and host of the request
// are recorded, as URLs of external systems can contain secrets (e.g. webhook URLs).
func ExternalHTTPTransport(rt http.RoundTripper) http.RoundTripper {
	if rt == nil {
		rt = http.DefaultTransport
	}
	return externalTransport{next: rt}
}

type externalTransport struct {
	next http.RoundTripper
}

func (t externalTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	ctx, span := otel.Tracer(scopeName).Start(r.Context(), r.Method+" "+r.URL.Host,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			semconv.HTTPRequestMethodKey.String(r.Method),
			semconv.ServerAddress(r.URL.Hostname()),
		))
	defer span.End()

	resp, err := t.next.RoundTrip(r.WithContext(ctx))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "request failed")
		return nil, err
	}

	span.SetAttributes(semconv.HTTPResponseStatusCode(resp.StatusCode))
	if resp.StatusCode >= http.StatusInternalServerError {
		span.SetStatus(codes.Error, resp.Status)
	}

	return resp, nil
}

// HTTPRoute provides a middleware that renames the active span to "<method> <route pattern>"
// and sets the http.route attribute once the request got routed.
// The route pattern is used instead of the raw path to keep span names low-cardinality.
// It has to be registered on the chi router that resolves the routes.
func HTTPRoute() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)

			span := trace.SpanFromContext(r.Context())
			if !span.IsRecording() {
				return
			}

			rctx := chi.RouteContext(r.Context())
			if rctx == nil {
				return
			}

			pattern := rctx.RoutePattern()
			if pattern == "" {
				return
			}

			span.SetName(r.Method + " " + pattern)
			span.SetAttributes(semconv.HTTPRoute(pattern))
		})
	}
}
