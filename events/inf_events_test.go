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
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/harness/gitness/stream"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

type infPayload struct {
	Name  string
	Count int
}

type infSent struct {
	streamID string
	payload  map[string]any
}

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

type infConsumer struct {
	handlers    map[string]stream.HandlerFunc
	registerErr error
	startErr    error
	errCh       chan error
	infoCh      chan string
	started     bool
	configured  int
}

func infNewConsumer() *infConsumer {
	return &infConsumer{
		handlers: map[string]stream.HandlerFunc{},
		errCh:    make(chan error, 8),
		infoCh:   make(chan string, 8),
	}
}

func (c *infConsumer) Register(streamID string, fn stream.HandlerFunc, _ ...stream.HandlerOption) error {
	if c.registerErr != nil {
		return c.registerErr
	}
	c.handlers[streamID] = fn
	return nil
}

func (c *infConsumer) Configure(opts ...stream.ConsumerOption) { c.configured += len(opts) }

func (c *infConsumer) Start(context.Context) error {
	c.started = true
	return c.startErr
}

func (c *infConsumer) Errors() <-chan error { return c.errCh }
func (c *infConsumer) Infos() <-chan string { return c.infoCh }

type infCollector struct {
	mx       sync.Mutex
	errors   int
	discards int
}

func (c *infCollector) RecordError(string, string) {
	c.mx.Lock()
	defer c.mx.Unlock()
	c.errors++
}

func (c *infCollector) RecordDiscard(string, string) {
	c.mx.Lock()
	defer c.mx.Unlock()
	c.discards++
}

func (c *infCollector) get() (int, int) {
	c.mx.Lock()
	defer c.mx.Unlock()
	return c.errors, c.discards
}

func infNewSystem(t *testing.T, consumer *infConsumer, producer *infProducer, coll Collector) *System {
	t.Helper()
	sys, err := NewSystem(func(string, string) (StreamConsumer, error) { return consumer, nil }, producer, coll)
	require.NoError(t, err)
	return sys
}

func infNewFactory(t *testing.T, sys *System) *ReaderFactory[*GenericReader] {
	t.Helper()
	f, err := NewReaderFactory(sys, "inf", func(r *GenericReader) (*GenericReader, error) { return r, nil })
	require.NoError(t, err)
	return f
}

func infEncode(t *testing.T, payload infPayload) []byte {
	t.Helper()
	buf := &bytes.Buffer{}
	require.NoError(t, gob.NewEncoder(buf).Encode(&Event[infPayload]{ID: "evt-1", Payload: payload}))
	return buf.Bytes()
}

func TestInfConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{"nil", nil, true},
		{"unsupported mode", &Config{Mode: "kafka", MaxStreamLength: 1}, true},
		{"empty mode", &Config{MaxStreamLength: 1}, true},
		{"zero length", &Config{Mode: ModeInMemory}, true},
		{"redis ok", &Config{Mode: ModeRedis, MaxStreamLength: 10}, false},
		{"inmemory ok", &Config{Mode: ModeInMemory, MaxStreamLength: 10}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestInfGetStreamID(t *testing.T) {
	require.Equal(t, "events:repo:created", getStreamID("repo", "created"))
}

func TestInfNewSystemValidation(t *testing.T) {
	consumerFn := func(string, string) (StreamConsumer, error) { return infNewConsumer(), nil }
	producer := &infProducer{}
	coll := NewNoopCollector()

	_, err := NewSystem(nil, producer, coll)
	require.Error(t, err)
	_, err = NewSystem(consumerFn, nil, coll)
	require.Error(t, err)
	_, err = NewSystem(consumerFn, producer, nil)
	require.Error(t, err)
	sys, err := NewSystem(consumerFn, producer, coll)
	require.NoError(t, err)

	_, err = NewReaderFactory(nil, "c", func(r *GenericReader) (*GenericReader, error) { return r, nil })
	require.Error(t, err)
	_, err = NewReaderFactory(sys, "", func(r *GenericReader) (*GenericReader, error) { return r, nil })
	require.Error(t, err)
	_, err = NewReaderFactory[*GenericReader](sys, "c", nil)
	require.Error(t, err)

	_, err = NewReporter(nil, "c")
	require.Error(t, err)
	_, err = NewReporter(sys, "")
	require.Error(t, err)
	rep, err := NewReporter(sys, "c")
	require.NoError(t, err)
	require.Equal(t, "c", rep.category)
}

func TestInfReporterSendEvent(t *testing.T) {
	ctx := context.Background()
	producer := &infProducer{}
	sys := infNewSystem(t, infNewConsumer(), producer, NewNoopCollector())
	rep, err := NewReporter(sys, "inf")
	require.NoError(t, err)

	id, err := ReporterSendEvent(rep, ctx, "evt", infPayload{Name: "a", Count: 2})
	require.NoError(t, err)
	require.NotEmpty(t, id)
	require.Len(t, producer.sent, 1)
	require.Equal(t, "events:inf:evt", producer.sent[0].streamID)

	raw, ok := producer.sent[0].payload[streamPayloadKey].([]byte)
	require.True(t, ok)
	var decoded Event[infPayload]
	require.NoError(t, gob.NewDecoder(bytes.NewReader(raw)).Decode(&decoded))
	require.Equal(t, id, decoded.ID)
	require.Equal(t, infPayload{Name: "a", Count: 2}, decoded.Payload)
	require.False(t, decoded.Timestamp.IsZero())

	producer.err = errors.New("send failed")
	_, err = ReporterSendEvent(rep, ctx, "evt", infPayload{})
	require.ErrorContains(t, err, "failed to send event")

	// gob can't encode interface values holding unregistered types
	producer.err = nil
	type infUnregistered struct{ X int }
	_, err = ReporterSendEvent(rep, ctx, "evt", struct{ V any }{V: infUnregistered{X: 1}})
	require.ErrorContains(t, err, "failed to encode payload")
}

func TestInfReaderRegisterEvent_HandlerBehaviour(t *testing.T) {
	ctx := context.Background()
	consumer := infNewConsumer()
	sys := infNewSystem(t, consumer, &infProducer{}, NewNoopCollector())

	var got *Event[infPayload]
	var handlerErr error
	_, err := infNewFactory(t, sys).Launch(ctx, "grp", "rdr", func(r *GenericReader) error {
		return ReaderRegisterEvent(r, "evt", func(_ context.Context, e *Event[infPayload]) error {
			got = e
			return handlerErr
		}, WithMaxRetries(3), WithIdleTimeout(time.Minute))
	})
	require.NoError(t, err)
	require.True(t, consumer.started)

	handler := consumer.handlers["events:inf:evt"]
	require.NotNil(t, handler)
	want := infPayload{Name: "n", Count: 5}

	t.Run("nil payload", func(t *testing.T) {
		require.ErrorContains(t, handler(ctx, "m1", nil), "stream payload is nil")
	})
	t.Run("missing event key", func(t *testing.T) {
		require.ErrorContains(t, handler(ctx, "m1", map[string]any{"other": 1}), "doesn't contain event")
	})
	t.Run("wrong type", func(t *testing.T) {
		require.ErrorContains(t, handler(ctx, "m1", map[string]any{streamPayloadKey: 42}), "not of expected type")
	})
	t.Run("undecodable", func(t *testing.T) {
		require.ErrorContains(t, handler(ctx, "m1", map[string]any{streamPayloadKey: "garbage"}), "can't be decoded")
	})
	t.Run("bytes payload", func(t *testing.T) {
		got = nil
		require.NoError(t, handler(ctx, "m1", map[string]any{streamPayloadKey: infEncode(t, want)}))
		require.Equal(t, want, got.Payload)
		require.Equal(t, "evt-1", got.ID)
	})
	t.Run("string payload as returned by redis", func(t *testing.T) {
		got = nil
		require.NoError(t, handler(ctx, "m1", map[string]any{streamPayloadKey: string(infEncode(t, want))}))
		require.Equal(t, want, got.Payload)
	})
	t.Run("handler error is returned", func(t *testing.T) {
		handlerErr = errors.New("handler failed")
		defer func() { handlerErr = nil }()
		err := handler(ctx, "m1", map[string]any{streamPayloadKey: infEncode(t, want)})
		require.ErrorIs(t, err, handlerErr)
	})
	t.Run("discard error is swallowed", func(t *testing.T) {
		handlerErr = NewDiscardEventErrorf("bad event %d", 1)
		defer func() { handlerErr = nil }()
		require.NoError(t, handler(ctx, "m1", map[string]any{streamPayloadKey: infEncode(t, want)}))
	})
}

func TestInfReaderRegisterEvent_RegisterError(t *testing.T) {
	consumer := infNewConsumer()
	consumer.registerErr = errors.New("register failed")
	sys := infNewSystem(t, consumer, &infProducer{}, NewNoopCollector())
	_, err := infNewFactory(t, sys).Launch(context.Background(), "g", "r", func(r *GenericReader) error {
		return ReaderRegisterEvent(r, "evt", func(context.Context, *Event[infPayload]) error { return nil })
	})
	require.ErrorContains(t, err, "failed custom setup of event reader")
	require.ErrorIs(t, err, consumer.registerErr)
}

func TestInfReaderFactory_LaunchErrors(t *testing.T) {
	ctx := context.Background()
	noop := func(*GenericReader) error { return nil }

	t.Run("validation", func(t *testing.T) {
		f := infNewFactory(t, infNewSystem(t, infNewConsumer(), &infProducer{}, NewNoopCollector()))
		_, err := f.Launch(ctx, "", "r", noop)
		require.Error(t, err)
		_, err = f.Launch(ctx, "g", "r", nil)
		require.Error(t, err)
	})
	t.Run("consumer factory error", func(t *testing.T) {
		sys, err := NewSystem(func(string, string) (StreamConsumer, error) { return nil, errors.New("nope") },
			&infProducer{}, NewNoopCollector())
		require.NoError(t, err)
		_, err = infNewFactory(t, sys).Launch(ctx, "g", "r", noop)
		require.ErrorContains(t, err, "failed to create new stream consumer")
	})
	t.Run("reader factory error", func(t *testing.T) {
		sys := infNewSystem(t, infNewConsumer(), &infProducer{}, NewNoopCollector())
		f, err := NewReaderFactory(sys, "inf", func(*GenericReader) (*GenericReader, error) {
			return nil, errors.New("reader boom")
		})
		require.NoError(t, err)
		_, err = f.Launch(ctx, "g", "r", noop)
		require.ErrorContains(t, err, "failed creation of event reader")
	})
	t.Run("setup error", func(t *testing.T) {
		f := infNewFactory(t, infNewSystem(t, infNewConsumer(), &infProducer{}, NewNoopCollector()))
		_, err := f.Launch(ctx, "g", "r", func(*GenericReader) error { return errors.New("setup boom") })
		require.ErrorContains(t, err, "failed custom setup of event reader")
	})
	t.Run("start error", func(t *testing.T) {
		consumer := infNewConsumer()
		consumer.startErr = errors.New("start boom")
		f := infNewFactory(t, infNewSystem(t, consumer, &infProducer{}, NewNoopCollector()))
		_, err := f.Launch(ctx, "g", "r", noop)
		require.ErrorContains(t, err, "failed to start consumer")
	})
}

func TestInfReaderFactory_LaunchConfiguresAndCancels(t *testing.T) {
	consumer := infNewConsumer()
	f := infNewFactory(t, infNewSystem(t, consumer, &infProducer{}, NewNoopCollector()))

	canceler, err := f.Launch(context.Background(), "g", "r", func(r *GenericReader) error {
		r.Configure(WithConcurrency(3), WithHandlerOptions(WithMaxRetries(1)))
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, 2, consumer.configured)

	require.NoError(t, canceler.Cancel())
	require.ErrorContains(t, canceler.Cancel(), "already been canceled")
}

func TestInfReaderFactory_RecordsErrorsAndDiscards(t *testing.T) {
	consumer := infNewConsumer()
	coll := &infCollector{}
	f := infNewFactory(t, infNewSystem(t, consumer, &infProducer{}, coll))

	canceler, err := f.Launch(context.Background(), "g", "r", func(*GenericReader) error { return nil })
	require.NoError(t, err)
	defer func() { _ = canceler.Cancel() }()

	consumer.errCh <- errors.New("plain failure")
	consumer.errCh <- &stream.DiscardedMessageError{MessageID: "1-0", StreamID: "s"}
	consumer.errCh <- errors.New("another failure")
	consumer.infoCh <- "some info"

	require.Eventually(t, func() bool {
		e, d := coll.get()
		return e == 2 && d == 1
	}, 5*time.Second, 5*time.Millisecond)
}

func TestInfDiscardEventError(t *testing.T) {
	inner := errors.New("inner")
	err := NewDiscardEventError(inner)
	require.ErrorIs(t, err, errDiscardEvent)
	require.ErrorIs(t, err, inner)
	require.Contains(t, err.Error(), "inner")
	require.NotErrorIs(t, inner, errDiscardEvent)

	errf := NewDiscardEventErrorf("value %d", 4)
	require.ErrorIs(t, errf, errDiscardEvent)
	require.Contains(t, errf.Error(), "value 4")
}

func TestInfNoopCollector(t *testing.T) {
	c := NewNoopCollector()
	c.RecordError("a", "b")
	c.RecordDiscard("a", "b")
	require.NotNil(t, ProvideNoopCollector())
}

func TestInfOptionsConversion(t *testing.T) {
	require.Empty(t, toStreamConsumerOptions(nil))
	require.Len(t, toStreamConsumerOptions([]ReaderOption{WithConcurrency(2), WithHandlerOptions()}), 2)
	require.Len(t, toStreamHandlerOptions([]HandlerOption{WithMaxRetries(1), WithIdleTimeout(time.Minute)}), 2)
}

func TestInfProvideSystem(t *testing.T) {
	t.Run("invalid config", func(t *testing.T) {
		_, err := ProvideSystem(Config{}, nil, NewNoopCollector())
		require.ErrorContains(t, err, "provided config is invalid")
	})
	t.Run("redis requires client", func(t *testing.T) {
		_, err := ProvideSystem(Config{Mode: ModeRedis, MaxStreamLength: 10}, nil, NewNoopCollector())
		require.ErrorContains(t, err, "redis client required")
	})
	t.Run("redis", func(t *testing.T) {
		client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
		defer client.Close()
		sys, err := ProvideSystem(Config{Mode: ModeRedis, Namespace: "ns", MaxStreamLength: 10},
			client, NewNoopCollector())
		require.NoError(t, err)
		consumer, err := sys.streamConsumerFactoryFn("grp", "cons")
		require.NoError(t, err)
		require.IsType(t, &stream.RedisConsumer{}, consumer)
		_, err = sys.streamConsumerFactoryFn("", "cons")
		require.Error(t, err)
	})
}

func TestInfProvideSystem_InMemoryRoundTrip(t *testing.T) {
	sys, err := ProvideSystem(Config{Mode: ModeInMemory, Namespace: "ns", MaxStreamLength: 10}, nil, NewNoopCollector())
	require.NoError(t, err)
	rep, err := NewReporter(sys, "inf")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	received := make(chan infPayload, 16)
	_, err = infNewFactory(t, sys).Launch(ctx, "grp", "rdr", func(r *GenericReader) error {
		return ReaderRegisterEvent(r, "evt", func(_ context.Context, e *Event[infPayload]) error {
			received <- e.Payload
			return nil
		})
	})
	require.NoError(t, err)

	// the in-memory consumer attaches to the broker asynchronously: keep sending until it receives.
	want := infPayload{Name: "roundtrip", Count: 9}
	require.Eventually(t, func() bool {
		_, sendErr := ReporterSendEvent(rep, ctx, "evt", want)
		require.NoError(t, sendErr)
		select {
		case got := <-received:
			require.Equal(t, want, got)
			return true
		case <-time.After(20 * time.Millisecond):
			return false
		}
	}, 5*time.Second, time.Millisecond)

	_, err = ProvideSystem(Config{Mode: ModeInMemory, MaxStreamLength: 1}, nil, NewNoopCollector())
	require.NoError(t, err)
}
