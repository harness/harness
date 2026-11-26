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

package events

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// The package tracer is bound to the first global tracer provider, so it is only set once per test binary.
var (
	recorder    = tracetest.NewSpanRecorder()
	setProvider sync.Once
	tp          = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
)

func TestEventTracing(t *testing.T) {
	setProvider.Do(func() { otel.SetTracerProvider(tp) })

	system, err := ProvideSystem(Config{Mode: ModeInMemory, Namespace: "test", MaxStreamLength: 100, Tracing: true}, nil,
		ProvideNoopCollector())
	require.NoError(t, err)

	reporter, err := NewReporter(system, "test")
	require.NoError(t, err)

	readerFactory, err := NewReaderFactory(system, "test", func(r *GenericReader) (*GenericReader, error) {
		return r, nil
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handled := make(chan trace.SpanContext, 1)
	_, err = readerFactory.Launch(ctx, "group", "reader", func(r *GenericReader) error {
		return ReaderRegisterEvent(r, "created", func(ctx context.Context, _ *Event[string]) error {
			handled <- trace.SpanContextFromContext(ctx)
			return nil
		})
	})
	require.NoError(t, err)

	sendCtx, parent := tp.Tracer("test").Start(ctx, "request")
	defer parent.End()

	// The in-memory broker discards events sent before the reader subscribed, so the event is resent until handled.
	var handlerSpan trace.SpanContext
	require.Eventually(t, func() bool {
		if _, err := ReporterSendEvent(reporter, sendCtx, "created", "payload"); !assert.NoError(t, err) {
			return false
		}

		select {
		case handlerSpan = <-handled:
			return true
		case <-time.After(50 * time.Millisecond):
			return false
		}
	}, 10*time.Second, time.Millisecond)

	assert.Equal(t, parent.SpanContext().TraceID(), handlerSpan.TraceID(), "handler should continue the sender's trace")

	var process sdktrace.ReadOnlySpan
	sends := map[trace.SpanID]sdktrace.ReadOnlySpan{}
	require.Eventually(t, func() bool {
		for _, span := range recorder.Ended() {
			if span.SpanContext().TraceID() != parent.SpanContext().TraceID() {
				continue
			}
			switch span.Name() {
			case "send events:test:created":
				sends[span.SpanContext().SpanID()] = span
			case "process events:test:created":
				process = span
			}
		}
		return process != nil && sends[process.Parent().SpanID()] != nil
	}, 10*time.Second, 10*time.Millisecond, "recorded: %v", spanNames(recorder.Ended()))

	send := sends[process.Parent().SpanID()]
	assert.Equal(t, trace.SpanKindProducer, send.SpanKind())
	assert.Equal(t, parent.SpanContext().SpanID(), send.Parent().SpanID())

	assert.Equal(t, trace.SpanKindConsumer, process.SpanKind())
	assert.Equal(t, handlerSpan.SpanID(), process.SpanContext().SpanID())
}

func TestNewSystemTraced(t *testing.T) {
	consumerFn, producer, err := provideSystemInMemory(Config{MaxStreamLength: 1})
	require.NoError(t, err)

	for _, traced := range []bool{false, true} {
		system, err := NewSystem(consumerFn, producer, ProvideNoopCollector(), traced)
		require.NoError(t, err)

		reporter, err := NewReporter(system, "test")
		require.NoError(t, err)
		assert.Equal(t, traced, reporter.traced)

		readerFactory, err := NewReaderFactory(system, "test", func(r *GenericReader) (*GenericReader, error) {
			return r, nil
		})
		require.NoError(t, err)
		assert.Equal(t, traced, readerFactory.traced)
	}
}

func spanNames(spans []sdktrace.ReadOnlySpan) []string {
	names := make([]string, len(spans))
	for i, span := range spans {
		names[i] = span.Name()
	}
	return names
}
