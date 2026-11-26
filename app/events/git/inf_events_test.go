// Copyright 2026 Harness, Inc.
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
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/harness/gitness/events"
	"github.com/harness/gitness/stream"

	"github.com/stretchr/testify/require"
)

type infSent struct {
	streamID string
	payload  map[string]any
}

// infProducer records everything sent through the events system.
type infProducer struct {
	mx   sync.Mutex
	sent []infSent
	err  error
}

func (p *infProducer) Send(_ context.Context, streamID string, payload map[string]any) (string, error) {
	p.mx.Lock()
	defer p.mx.Unlock()
	if p.err != nil {
		return "", p.err
	}
	p.sent = append(p.sent, infSent{streamID: streamID, payload: payload})
	return "1-0", nil
}

func (p *infProducer) count() int {
	p.mx.Lock()
	defer p.mx.Unlock()
	return len(p.sent)
}

func (p *infProducer) last(t *testing.T) infSent {
	t.Helper()
	p.mx.Lock()
	defer p.mx.Unlock()
	require.NotEmpty(t, p.sent)
	return p.sent[len(p.sent)-1]
}

// infConsumer records the handlers registered through the reader.
type infConsumer struct {
	handlers map[string]stream.HandlerFunc
	started  bool
	opts     int
}

func (c *infConsumer) Register(streamID string, fn stream.HandlerFunc, _ ...stream.HandlerOption) error {
	c.handlers[streamID] = fn
	return nil
}

func (c *infConsumer) Configure(opts ...stream.ConsumerOption) { c.opts += len(opts) }

func (c *infConsumer) Start(context.Context) error {
	c.started = true
	return nil
}

func (c *infConsumer) Errors() <-chan error {
	ch := make(chan error)
	close(ch)
	return ch
}

func (c *infConsumer) Infos() <-chan string {
	ch := make(chan string)
	close(ch)
	return ch
}

type infHarness struct {
	producer *infProducer
	consumer *infConsumer
	system   *events.System
	reporter *Reporter
}

func infNewHarness(t *testing.T) *infHarness {
	t.Helper()
	h := &infHarness{
		producer: &infProducer{},
		consumer: &infConsumer{handlers: map[string]stream.HandlerFunc{}},
	}
	var err error
	h.system, err = events.NewSystem(
		func(string, string) (events.StreamConsumer, error) { return h.consumer, nil },
		h.producer,
		events.NewNoopCollector(),
	)
	require.NoError(t, err)
	h.reporter, err = NewReporter(h.system)
	require.NoError(t, err)
	return h
}

// launch starts a reader whose handlers are registered by setup.
func (h *infHarness) launch(ctx context.Context, t *testing.T, setup func(*Reader) error) {
	t.Helper()
	factory, err := NewReaderFactory(h.system)
	require.NoError(t, err)
	canceler, err := factory.Launch(ctx, "grp", "reader", func(r *Reader) error {
		r.Configure(events.WithConcurrency(1))
		return setup(r)
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = canceler.Cancel() })
	require.True(t, h.consumer.started)
	require.Equal(t, 1, h.consumer.opts)
}

// deliver feeds a sent message to the handler registered for its stream.
func (h *infHarness) deliver(ctx context.Context, t *testing.T, sent infSent) {
	t.Helper()
	handler, ok := h.consumer.handlers[sent.streamID]
	require.True(t, ok, "no handler registered for stream %q", sent.streamID)
	require.NoError(t, handler(ctx, "1-0", sent.payload))
}

// infFill sets every exported field to a deterministic non-zero value.
func infFill(v reflect.Value) {
	switch v.Kind() { //nolint:exhaustive // only the kinds used by event payloads are filled.
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(7)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(7)
	case reflect.Float32, reflect.Float64:
		v.SetFloat(1.5)
	case reflect.Bool:
		v.SetBool(true)
	case reflect.String:
		v.SetString("inf-value")
	case reflect.Ptr:
		v.Set(reflect.New(v.Type().Elem()))
		infFill(v.Elem())
	case reflect.Slice:
		s := reflect.MakeSlice(v.Type(), 2, 2)
		for i := 0; i < s.Len(); i++ {
			infFill(s.Index(i))
		}
		v.Set(s)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Field(i).CanSet() {
				infFill(v.Field(i))
			}
		}
	default:
		// interfaces, maps, etc. keep their zero value.
	}
}

func TestInfEvents_RoundTrip(t *testing.T) {
	ctx := context.Background()

	t.Run("BranchCreated", func(t *testing.T) {
		h := infNewHarness(t)
		want := &BranchCreatedPayload{}
		infFill(reflect.ValueOf(want).Elem())
		var got *BranchCreatedPayload
		h.launch(ctx, t, func(r *Reader) error {
			return r.RegisterBranchCreated(func(_ context.Context, e *events.Event[*BranchCreatedPayload]) error {
				got = e.Payload
				return nil
			})
		})

		h.reporter.BranchCreated(ctx, want)
		sent := h.producer.last(t)
		require.Equal(t, "events:git:branch-created", sent.streamID)
		h.deliver(ctx, t, sent)
		require.Equal(t, want, got)

		before := h.producer.count()
		// failures of the producer are handled.
		h.producer.err = errors.New("send failed")
		h.reporter.BranchCreated(ctx, want)
		require.Equal(t, before, h.producer.count())
	})

	t.Run("BranchUpdated", func(t *testing.T) {
		h := infNewHarness(t)
		want := &BranchUpdatedPayload{}
		infFill(reflect.ValueOf(want).Elem())
		var got *BranchUpdatedPayload
		h.launch(ctx, t, func(r *Reader) error {
			return r.RegisterBranchUpdated(func(_ context.Context, e *events.Event[*BranchUpdatedPayload]) error {
				got = e.Payload
				return nil
			})
		})

		h.reporter.BranchUpdated(ctx, want)
		sent := h.producer.last(t)
		require.Equal(t, "events:git:branch-updated", sent.streamID)
		h.deliver(ctx, t, sent)
		require.Equal(t, want, got)

		before := h.producer.count()
		// failures of the producer are handled.
		h.producer.err = errors.New("send failed")
		h.reporter.BranchUpdated(ctx, want)
		require.Equal(t, before, h.producer.count())
	})

	t.Run("BranchDeleted", func(t *testing.T) {
		h := infNewHarness(t)
		want := &BranchDeletedPayload{}
		infFill(reflect.ValueOf(want).Elem())
		var got *BranchDeletedPayload
		h.launch(ctx, t, func(r *Reader) error {
			return r.RegisterBranchDeleted(func(_ context.Context, e *events.Event[*BranchDeletedPayload]) error {
				got = e.Payload
				return nil
			})
		})

		h.reporter.BranchDeleted(ctx, want)
		sent := h.producer.last(t)
		require.Equal(t, "events:git:branch-deleted", sent.streamID)
		h.deliver(ctx, t, sent)
		require.Equal(t, want, got)

		before := h.producer.count()
		// failures of the producer are handled.
		h.producer.err = errors.New("send failed")
		h.reporter.BranchDeleted(ctx, want)
		require.Equal(t, before, h.producer.count())
	})

	t.Run("TagCreated", func(t *testing.T) {
		h := infNewHarness(t)
		want := &TagCreatedPayload{}
		infFill(reflect.ValueOf(want).Elem())
		var got *TagCreatedPayload
		h.launch(ctx, t, func(r *Reader) error {
			return r.RegisterTagCreated(func(_ context.Context, e *events.Event[*TagCreatedPayload]) error {
				got = e.Payload
				return nil
			})
		})

		h.reporter.TagCreated(ctx, want)
		sent := h.producer.last(t)
		require.Equal(t, "events:git:tag-created", sent.streamID)
		h.deliver(ctx, t, sent)
		require.Equal(t, want, got)

		before := h.producer.count()
		// failures of the producer are handled.
		h.producer.err = errors.New("send failed")
		h.reporter.TagCreated(ctx, want)
		require.Equal(t, before, h.producer.count())
	})

	t.Run("TagUpdated", func(t *testing.T) {
		h := infNewHarness(t)
		want := &TagUpdatedPayload{}
		infFill(reflect.ValueOf(want).Elem())
		var got *TagUpdatedPayload
		h.launch(ctx, t, func(r *Reader) error {
			return r.RegisterTagUpdated(func(_ context.Context, e *events.Event[*TagUpdatedPayload]) error {
				got = e.Payload
				return nil
			})
		})

		h.reporter.TagUpdated(ctx, want)
		sent := h.producer.last(t)
		require.Equal(t, "events:git:tag-updated", sent.streamID)
		h.deliver(ctx, t, sent)
		require.Equal(t, want, got)

		before := h.producer.count()
		// failures of the producer are handled.
		h.producer.err = errors.New("send failed")
		h.reporter.TagUpdated(ctx, want)
		require.Equal(t, before, h.producer.count())
	})

	t.Run("TagDeleted", func(t *testing.T) {
		h := infNewHarness(t)
		want := &TagDeletedPayload{}
		infFill(reflect.ValueOf(want).Elem())
		var got *TagDeletedPayload
		h.launch(ctx, t, func(r *Reader) error {
			return r.RegisterTagDeleted(func(_ context.Context, e *events.Event[*TagDeletedPayload]) error {
				got = e.Payload
				return nil
			})
		})

		h.reporter.TagDeleted(ctx, want)
		sent := h.producer.last(t)
		require.Equal(t, "events:git:tag-deleted", sent.streamID)
		h.deliver(ctx, t, sent)
		require.Equal(t, want, got)

		before := h.producer.count()
		// failures of the producer are handled.
		h.producer.err = errors.New("send failed")
		h.reporter.TagDeleted(ctx, want)
		require.Equal(t, before, h.producer.count())
	})
}

func TestInfEvents_ReaderFactoryInvalidSystem(t *testing.T) {
	_, err := NewReaderFactory(nil)
	require.Error(t, err)
	_, err = NewReporter(nil)
	require.Error(t, err)
}
