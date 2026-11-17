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

// Package tracing sets up OpenTelemetry tracing.
//
// Tracing is turned on through the service's configuration (Config), which services load from the standard
// OTEL_TRACES_EXPORTER environment variable; what gets traced is switched per area by the services. The exporter,
// sampler and resource are configured through the standard OpenTelemetry environment variables, e.g.:
//   - OTEL_EXPORTER_OTLP_ENDPOINT / OTEL_EXPORTER_OTLP_TRACES_ENDPOINT (default https://localhost:4317)
//   - OTEL_EXPORTER_OTLP_INSECURE
//   - OTEL_TRACES_SAMPLER / OTEL_TRACES_SAMPLER_ARG (default parentbased_always_on)
//   - OTEL_SERVICE_NAME (overrides the service name passed to Init)
//   - OTEL_RESOURCE_ATTRIBUTES
package tracing

import (
	"context"
	"fmt"
	"strings"

	"github.com/harness/gitness/version"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

const (
	// ExporterOTLP exports spans with OTLP over gRPC.
	ExporterOTLP = "otlp"
	// ExporterNone keeps tracing off. Services default to it instead of the OpenTelemetry default "otlp".
	ExporterNone = "none"
)

// Config is the tracing configuration of a service.
type Config struct {
	// Exporter is the trace exporter (OTEL_TRACES_EXPORTER), ExporterOTLP or ExporterNone.
	Exporter string
}

// Enabled reports whether the exporter turns tracing on.
func Enabled(exporter string) bool {
	return strings.EqualFold(exporter, ExporterOTLP)
}

// ShutdownFunc flushes pending spans and releases the exporter.
type ShutdownFunc func(context.Context) error

// Init configures the global tracer provider and propagator for the service.
// If tracing is disabled, the global no-op tracer provider is kept and the returned ShutdownFunc does nothing.
func Init(ctx context.Context, serviceName string, config Config) (ShutdownFunc, error) {
	// W3C trace context is propagated even if tracing is disabled to not break traces of upstream services.
	otel.SetTextMapPropagator(propagator)

	if !Enabled(config.Exporter) {
		if config.Exporter != "" && !strings.EqualFold(config.Exporter, ExporterNone) {
			log.Ctx(ctx).Warn().Str("exporter", config.Exporter).Msg("unsupported trace exporter, tracing stays off")
		}
		log.Ctx(ctx).Info().Msg("opentelemetry tracing is disabled")
		return func(context.Context) error { return nil }, nil
	}

	exporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create otlp trace exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(version.Version.String()),
		),
		resource.WithHost(),
		resource.WithTelemetrySDK(),
		resource.WithFromEnv(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create otel resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		log.Warn().Err(err).Msg("opentelemetry error")
	}))

	log.Ctx(ctx).Info().Str("service_name", serviceName).Msg("opentelemetry tracing is enabled")

	return tp.Shutdown, nil
}
