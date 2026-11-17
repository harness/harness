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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func resetGlobals(t *testing.T) {
	t.Cleanup(func() {
		otel.SetTracerProvider(noop.NewTracerProvider())
	})
}

func TestInitDisabled(t *testing.T) {
	for _, exporter := range []string{"", ExporterNone, "console"} {
		t.Run("exporter="+exporter, func(t *testing.T) {
			resetGlobals(t)

			shutdown, err := Init(context.Background(), "test-service", Config{Exporter: exporter})
			require.NoError(t, err)
			require.NoError(t, shutdown(context.Background()))

			_, isSDK := otel.GetTracerProvider().(*sdktrace.TracerProvider)
			assert.False(t, isSDK, "sdk tracer provider must not be installed when tracing is disabled")
		})
	}
}

func TestInitEnabled(t *testing.T) {
	resetGlobals(t)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:4317")

	shutdown, err := Init(context.Background(), "test-service", Config{Exporter: "OTLP"})
	require.NoError(t, err)

	tp, isSDK := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	require.True(t, isSDK, "sdk tracer provider must be installed when tracing is enabled")

	_, span := tp.Tracer("test").Start(context.Background(), "test")
	assert.True(t, span.IsRecording())
	span.End()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	// exporting fails without a collector, only make sure shutdown returns.
	_ = shutdown(ctx)
}
