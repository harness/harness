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

package server

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestProvideRedis(t *testing.T) {
	cfg := &types.Config{}
	cfg.Redis.Endpoint = "127.0.0.1:6379"
	cfg.Tracing.Exporter = "otlp"
	cfg.Tracing.RedisEnabled = true

	client, err := ProvideRedis(cfg)
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.NoError(t, client.Close())

	cfg.Tracing.RedisEnabled = false
	untracedClient, err := ProvideRedis(cfg)
	require.NoError(t, err)
	require.NotNil(t, untracedClient)
	assert.NoError(t, untracedClient.Close())

	cfg.Redis.SSLEnabled = true
	cfg.Redis.CertPath = filepath.Join(t.TempDir(), "missing-ca.pem")
	_, err = ProvideRedis(cfg)
	assert.Error(t, err)
}

func TestNewRedisClientTracing(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { otel.SetTracerProvider(previous) })

	cfg := &types.Config{}
	// Nothing listens on port 1, so commands fail fast but still pass through the hooks.
	cfg.Redis.Endpoint = "127.0.0.1:1"
	cfg.Redis.MaxRetries = -1

	// The tracing hook only records commands issued under a recording span.
	ctx, parent := tp.Tracer("test").Start(context.Background(), "request")
	for _, traced := range []bool{false, true} {
		client, err := newRedisClient(cfg, traced)
		require.NoError(t, err)
		assert.Error(t, client.Ping(ctx).Err())
		require.NoError(t, client.Close())
	}
	parent.End()

	var commands []string
	for _, span := range recorder.Ended() {
		if span.Parent().SpanID() == parent.SpanContext().SpanID() {
			commands = append(commands, span.Name())
		}
	}
	assert.Equal(t, []string{"ping"}, commands, "only the traced client should record its commands")
}

func TestProvideEventsConfigTracing(t *testing.T) {
	cfg := &types.Config{}
	assert.False(t, ProvideEventsConfig(cfg).Tracing)

	cfg.Tracing.Exporter = "otlp"
	assert.False(t, ProvideEventsConfig(cfg).Tracing)

	cfg.Tracing.RedisEnabled = true
	assert.True(t, ProvideEventsConfig(cfg).Tracing)
}
