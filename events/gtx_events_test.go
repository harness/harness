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
	"sync"
	"testing"
	"time"

	"github.com/harness/gitness/stream"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	gtxCategory  = "gtx-category"
	gtxEventType = EventType("gtx-created")
	gtxTimeout   = 5 * time.Second
)

type gtxPayload struct {
	Name  string
	Count int
}

// gtxCollector records the calls to the collector.
type gtxCollector struct {
	mu       sync.Mutex
	errors   []string
	discards []string
}

func (c *gtxCollector) RecordError(category, group string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errors = append(c.errors, category+"/"+group)
}

func (c *gtxCollector) RecordDiscard(category, group string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.discards = append(c.discards, category+"/"+group)
}

func (c *gtxCollector) errorCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.errors)
}

func gtxNewSystem(t *testing.T, collector Collector) *System {
	t.Helper()

	system, err := ProvideSystem(Config{
		Mode:            ModeInMemory,
		Namespace:       "gtx",
		MaxStreamLength: 100,
	}, nil, collector)
	require.NoError(t, err)

	return system
}

func gtxNewReaderFactory(t *testing.T, system *System) *ReaderFactory[*GenericReader] {
	t.Helper()

	factory, err := NewReaderFactory(system, gtxCategory, func(r *GenericReader) (*GenericReader, error) {
		return r, nil
	})
	require.NoError(t, err)

	return factory
}

// gtxSendUntil sends events until the condition is met. Events sent before the reader's queue is
// created are discarded by the in-memory broker, hence the sending is repeated.
func gtxSendUntil(t *testing.T, send func() error, done <-chan struct{}) {
	t.Helper()

	deadline := time.After(gtxTimeout)
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	for {
		require.NoError(t, send())

		select {
		case <-done:
			return
		case <-deadline:
			t.Fatal("event wasn't received in time")
		case <-ticker.C:
		}
	}
}

func TestGtxConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr string
	}{
		{name: "nil", config: nil, wantErr: "config is required"},
		{name: "in-memory", config: &Config{Mode: ModeInMemory, MaxStreamLength: 1}},
		{name: "redis", config: &Config{Mode: ModeRedis, MaxStreamLength: 1000}},
		{name: "unsupported mode", config: &Config{Mode: "kafka", MaxStreamLength: 1}, wantErr: "'kafka' is not supported"},
		{name: "empty mode", config: &Config{MaxStreamLength: 1}, wantErr: "is not supported"},
		{name: "zero stream length", config: &Config{Mode: ModeInMemory}, wantErr: "positive number"},
		{
			name: "negative stream length", config: &Config{Mode: ModeInMemory, MaxStreamLength: -1},
			wantErr: "positive number",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.config.Validate()
			if test.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestGtxProvideSystem(t *testing.T) {
	t.Run("invalid config", func(t *testing.T) {
		_, err := ProvideSystem(Config{}, nil, NewNoopCollector())
		require.ErrorContains(t, err, "provided config is invalid")
	})

	t.Run("in-memory", func(t *testing.T) {
		system, err := ProvideSystem(Config{Mode: ModeInMemory, MaxStreamLength: 10}, nil, ProvideNoopCollector())
		require.NoError(t, err)
		require.NotNil(t, system)
		assert.NotNil(t, system.streamProducer)
		assert.NotNil(t, system.streamConsumerFactoryFn)
	})

	t.Run("redis requires a client", func(t *testing.T) {
		_, err := ProvideSystem(Config{Mode: ModeRedis, MaxStreamLength: 10}, nil, NewNoopCollector())
		require.ErrorContains(t, err, "redis client required")
		require.ErrorContains(t, err, "failed to setup event system for mode 'redis'")
	})

	t.Run("redis with a client", func(t *testing.T) {
		// the client connects lazily, so no redis server is needed to build the system.
		client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
		t.Cleanup(func() { _ = client.Close() })

		system, err := ProvideSystem(Config{
			Mode:            ModeRedis,
			Namespace:       "gtx",
			MaxStreamLength: 10,
			ProducerRetry:   stream.ProducerRetryConfig{},
		}, client, NewNoopCollector())
		require.NoError(t, err)
		require.NotNil(t, system)

		consumer, err := system.streamConsumerFactoryFn("group", "consumer")
		require.NoError(t, err)
		assert.NotNil(t, consumer)

		_, err = system.streamConsumerFactoryFn("", "consumer")
		require.Error(t, err)
	})
}

func TestGtxNewSystem_Validation(t *testing.T) {
	consumerFn := func(string, string) (StreamConsumer, error) { return newGtxFakeConsumer(), nil }
	producer := stream.NewMemoryProducer(nil, "ns")
	collector := NewNoopCollector()

	_, err := NewSystem(nil, producer, collector, false)
	require.ErrorContains(t, err, "streamConsumerFactoryFunc")

	_, err = NewSystem(consumerFn, nil, collector, false)
	require.ErrorContains(t, err, "streamProducer")

	_, err = NewSystem(consumerFn, producer, nil, false)
	require.ErrorContains(t, err, "collector")

	system, err := NewSystem(consumerFn, producer, collector, false)
	require.NoError(t, err)
	assert.NotNil(t, system)
}

func TestGtxNewReaderFactoryAndReporter_Validation(t *testing.T) {
	system := gtxNewSystem(t, NewNoopCollector())
	fn := func(r *GenericReader) (*GenericReader, error) { return r, nil }

	_, err := NewReaderFactory(nil, gtxCategory, fn)
	require.ErrorContains(t, err, "system")

	_, err = NewReaderFactory(system, "", fn)
	require.ErrorContains(t, err, "category")

	_, err = NewReaderFactory[*GenericReader](system, gtxCategory, nil)
	require.ErrorContains(t, err, "fn")

	factory, err := NewReaderFactory(system, gtxCategory, fn)
	require.NoError(t, err)
	assert.Equal(t, gtxCategory, factory.category)

	_, err = NewReporter(nil, gtxCategory)
	require.ErrorContains(t, err, "system")

	_, err = NewReporter(system, "")
	require.ErrorContains(t, err, "category")

	reporter, err := NewReporter(system, gtxCategory)
	require.NoError(t, err)
	assert.Equal(t, gtxCategory, reporter.category)
}

func TestGtxEvents_EndToEnd(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	system := gtxNewSystem(t, NewNoopCollector())
	factory := gtxNewReaderFactory(t, system)

	reporter, err := NewReporter(system, gtxCategory)
	require.NoError(t, err)

	received := make(chan *Event[gtxPayload], 16)
	canceler, err := factory.Launch(ctx, "group", "reader", func(r *GenericReader) error {
		r.Configure(WithConcurrency(1), WithHandlerOptions(WithMaxRetries(1), WithIdleTimeout(time.Minute)))

		return ReaderRegisterEvent(r, gtxEventType, func(_ context.Context, e *Event[gtxPayload]) error {
			received <- e
			return nil
		}, WithMaxRetries(3), WithIdleTimeout(time.Minute))
	})
	require.NoError(t, err)

	var (
		mu      sync.Mutex
		sentIDs []string
	)
	done := make(chan struct{})
	go func() {
		<-received
		close(done)
	}()

	gtxSendUntil(t, func() error {
		id, err := ReporterSendEvent(reporter, ctx, gtxEventType, gtxPayload{Name: "first", Count: 42})
		if err != nil {
			return err
		}
		mu.Lock()
		sentIDs = append(sentIDs, id)
		mu.Unlock()
		return nil
	}, done)

	require.NoError(t, canceler.Cancel())
	require.ErrorContains(t, canceler.Cancel(), "already been canceled")

	mu.Lock()
	defer mu.Unlock()
	assert.NotEmpty(t, sentIDs)
}

func TestGtxEvents_HandlerResults(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	collector := &gtxCollector{}
	system := gtxNewSystem(t, collector)
	factory := gtxNewReaderFactory(t, system)

	reporter, err := NewReporter(system, gtxCategory)
	require.NoError(t, err)

	discarded := make(chan struct{}, 16)
	failed := make(chan struct{}, 16)

	_, err = factory.Launch(ctx, "group", "reader", func(r *GenericReader) error {
		require.NoError(t, ReaderRegisterEvent(r, "gtx-discard",
			func(context.Context, *Event[gtxPayload]) error {
				discarded <- struct{}{}
				return NewDiscardEventErrorf("gtx discard %d", 1)
			}))

		return ReaderRegisterEvent(r, "gtx-fail",
			func(context.Context, *Event[gtxPayload]) error {
				failed <- struct{}{}
				return errors.New("gtx failure")
			}, WithIdleTimeout(time.Hour))
	})
	require.NoError(t, err)

	t.Run("discarded events don't cause errors", func(t *testing.T) {
		sent := make(chan struct{})
		go func() {
			<-discarded
			close(sent)
		}()

		gtxSendUntil(t, func() error {
			_, err := ReporterSendEvent(reporter, ctx, "gtx-discard", gtxPayload{})
			return err
		}, sent)

		time.Sleep(100 * time.Millisecond)
		assert.Zero(t, collector.errorCount())
	})

	t.Run("failing events are recorded", func(t *testing.T) {
		sent := make(chan struct{})
		go func() {
			<-failed
			close(sent)
		}()

		gtxSendUntil(t, func() error {
			_, err := ReporterSendEvent(reporter, ctx, "gtx-fail", gtxPayload{})
			return err
		}, sent)

		require.Eventually(t, func() bool { return collector.errorCount() > 0 }, gtxTimeout, 10*time.Millisecond)

		collector.mu.Lock()
		defer collector.mu.Unlock()
		assert.Equal(t, gtxCategory+"/group", collector.errors[0])
	})
}

// gtxFakeConsumer is a stream consumer with configurable failures.
type gtxFakeConsumer struct {
	registerErr error
	startErr    error
	errorCh     chan error
	infoCh      chan string
	registered  map[string]stream.HandlerFunc
	configured  int
	started     bool
}

func newGtxFakeConsumer() *gtxFakeConsumer {
	return &gtxFakeConsumer{
		errorCh:    make(chan error, 4),
		infoCh:     make(chan string, 4),
		registered: map[string]stream.HandlerFunc{},
	}
}

func (c *gtxFakeConsumer) Register(streamID string, handler stream.HandlerFunc, _ ...stream.HandlerOption) error {
	if c.registerErr != nil {
		return c.registerErr
	}
	c.registered[streamID] = handler
	return nil
}

func (c *gtxFakeConsumer) Configure(...stream.ConsumerOption) { c.configured++ }

func (c *gtxFakeConsumer) Start(context.Context) error {
	c.started = c.startErr == nil
	return c.startErr
}

func (c *gtxFakeConsumer) Errors() <-chan error { return c.errorCh }
func (c *gtxFakeConsumer) Infos() <-chan string { return c.infoCh }

func TestGtxReaderFactory_Launch(t *testing.T) {
	ctx := context.Background()

	newFactory := func(t *testing.T, consumer *gtxFakeConsumer, consumerErr error, collector Collector,
		readerFn ReaderFactoryFunc[*GenericReader],
	) *ReaderFactory[*GenericReader] {
		t.Helper()

		system, err := NewSystem(
			func(string, string) (StreamConsumer, error) {
				if consumerErr != nil {
					return nil, consumerErr
				}
				return consumer, nil
			},
			stream.NewMemoryProducer(nil, "ns"),
			collector,
			false,
		)
		require.NoError(t, err)

		factory, err := NewReaderFactory(system, gtxCategory, readerFn)
		require.NoError(t, err)

		return factory
	}
	identity := func(r *GenericReader) (*GenericReader, error) { return r, nil }
	noopSetup := func(*GenericReader) error { return nil }

	t.Run("group name is required", func(t *testing.T) {
		_, err := newFactory(t, newGtxFakeConsumer(), nil, NewNoopCollector(), identity).Launch(ctx, "", "r", noopSetup)
		require.ErrorContains(t, err, "groupName can't be empty")
	})

	t.Run("setup is required", func(t *testing.T) {
		_, err := newFactory(t, newGtxFakeConsumer(), nil, NewNoopCollector(), identity).Launch(ctx, "g", "r", nil)
		require.ErrorContains(t, err, "setup function can't be nil")
	})

	t.Run("consumer creation fails", func(t *testing.T) {
		f := newFactory(t, nil, errors.New("gtx consumer failure"), NewNoopCollector(), identity)
		_, err := f.Launch(ctx, "g", "r", noopSetup)
		require.ErrorContains(t, err, "failed to create new stream consumer: gtx consumer failure")
	})

	t.Run("reader creation fails", func(t *testing.T) {
		f := newFactory(t, newGtxFakeConsumer(), nil, NewNoopCollector(),
			func(*GenericReader) (*GenericReader, error) { return nil, errors.New("gtx reader failure") })
		_, err := f.Launch(ctx, "g", "r", noopSetup)
		require.ErrorContains(t, err, "failed creation of event reader")
		require.ErrorContains(t, err, "gtx reader failure")
	})

	t.Run("setup fails", func(t *testing.T) {
		consumer := newGtxFakeConsumer()
		f := newFactory(t, consumer, nil, NewNoopCollector(), identity)
		_, err := f.Launch(ctx, "g", "r", func(*GenericReader) error { return errors.New("gtx setup failure") })
		require.ErrorContains(t, err, "failed custom setup of event reader: gtx setup failure")
		assert.False(t, consumer.started)
	})

	t.Run("start fails", func(t *testing.T) {
		consumer := newGtxFakeConsumer()
		consumer.startErr = errors.New("gtx start failure")
		f := newFactory(t, consumer, nil, NewNoopCollector(), identity)

		canceler, err := f.Launch(ctx, "g", "r", noopSetup)
		require.ErrorContains(t, err, "failed to start consumer: gtx start failure")
		assert.Nil(t, canceler)
	})

	t.Run("register and configure are forwarded to the consumer", func(t *testing.T) {
		consumer := newGtxFakeConsumer()
		f := newFactory(t, consumer, nil, NewNoopCollector(), identity)

		canceler, err := f.Launch(ctx, "g", "r", func(r *GenericReader) error {
			r.Configure(WithConcurrency(2))
			return ReaderRegisterEvent(r, gtxEventType, func(context.Context, *Event[gtxPayload]) error { return nil })
		})
		require.NoError(t, err)
		require.NoError(t, canceler.Cancel())

		assert.True(t, consumer.started)
		assert.Equal(t, 1, consumer.configured)
		assert.Contains(t, consumer.registered, "events:"+gtxCategory+":"+string(gtxEventType))
	})

	t.Run("register failure is returned from the setup helper", func(t *testing.T) {
		consumer := newGtxFakeConsumer()
		consumer.registerErr = errors.New("gtx register failure")
		f := newFactory(t, consumer, nil, NewNoopCollector(), identity)

		_, err := f.Launch(ctx, "g", "r", func(r *GenericReader) error {
			return ReaderRegisterEvent(r, gtxEventType, func(context.Context, *Event[gtxPayload]) error { return nil })
		})
		require.ErrorContains(t, err, "gtx register failure")
	})

	t.Run("consumer errors and infos are consumed", func(t *testing.T) {
		consumer := newGtxFakeConsumer()
		collector := &gtxCollector{}
		f := newFactory(t, consumer, nil, collector, identity)

		canceler, err := f.Launch(ctx, "g", "r", noopSetup)
		require.NoError(t, err)
		t.Cleanup(func() { _ = canceler.Cancel() })

		consumer.errorCh <- errors.New("gtx plain error")
		consumer.errorCh <- &stream.DiscardedMessageError{MessageID: "m", StreamID: "s"}
		consumer.infoCh <- "gtx info"

		require.Eventually(t, func() bool {
			collector.mu.Lock()
			defer collector.mu.Unlock()
			return len(collector.errors) == 1 && len(collector.discards) == 1
		}, gtxTimeout, 10*time.Millisecond)

		collector.mu.Lock()
		defer collector.mu.Unlock()
		assert.Equal(t, []string{gtxCategory + "/g"}, collector.errors)
		assert.Equal(t, []string{gtxCategory + "/g"}, collector.discards)
	})
}

func TestGtxReaderCanceler_CancelFailure(t *testing.T) {
	c := &ReaderCanceler{cancelFn: func() error { return errors.New("gtx cancel failure") }}

	err := c.Cancel()
	require.ErrorContains(t, err, "failed to cancel reader: gtx cancel failure")

	// a failed cancel doesn't mark the reader as canceled, so it can be retried.
	require.ErrorContains(t, c.Cancel(), "gtx cancel failure")
}

func TestGtxReaderRegisterEvent_PayloadHandling(t *testing.T) {
	ctx := context.Background()

	consumer := newGtxFakeConsumer()
	reader := &GenericReader{streamConsumer: consumer, category: gtxCategory}

	called := 0
	require.NoError(t, ReaderRegisterEvent(reader, gtxEventType,
		func(context.Context, *Event[gtxPayload]) error {
			called++
			return nil
		}))

	handler := consumer.registered[getStreamID(gtxCategory, gtxEventType)]
	require.NotNil(t, handler)

	// produce a valid gob payload with the reporter.
	captured := &gtxCapturingProducer{}
	reporter := &GenericReporter{producer: captured, category: gtxCategory}
	_, err := ReporterSendEvent(reporter, ctx, gtxEventType, gtxPayload{Name: "n", Count: 1})
	require.NoError(t, err)
	validBytes, ok := captured.payload[streamPayloadKey].([]byte)
	require.True(t, ok)
	valid := any(validBytes)

	tests := []struct {
		name    string
		payload map[string]any
		wantErr string
		called  int
	}{
		{name: "bytes payload", payload: map[string]any{streamPayloadKey: valid}, called: 1},
		{name: "string payload", payload: map[string]any{streamPayloadKey: string(validBytes)}, called: 1},
		{name: "nil payload", payload: nil, wantErr: "stream payload is nil"},
		{name: "missing event key", payload: map[string]any{"other": 1}, wantErr: "doesn't contain event"},
		{name: "unexpected type", payload: map[string]any{streamPayloadKey: 12}, wantErr: "not of expected type"},
		{name: "undecodable", payload: map[string]any{streamPayloadKey: []byte("garbage")}, wantErr: "can't be decoded"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			called = 0
			err := handler(ctx, "msg-1", test.payload)
			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, test.called, called)
		})
	}
}

type gtxCapturingProducer struct {
	payload map[string]any
	err     error
}

func (p *gtxCapturingProducer) Send(_ context.Context, _ string, payload map[string]any) (string, error) {
	p.payload = payload
	return "id", p.err
}

func TestGtxReporterSendEvent_Failures(t *testing.T) {
	ctx := context.Background()

	t.Run("producer failure", func(t *testing.T) {
		producer := &gtxCapturingProducer{err: errors.New("gtx send failure")}
		reporter := &GenericReporter{producer: producer, category: gtxCategory}

		_, err := ReporterSendEvent(reporter, ctx, gtxEventType, gtxPayload{})
		require.ErrorContains(t, err, "failed to send event: gtx send failure")
	})

	t.Run("payload that can't be encoded", func(t *testing.T) {
		reporter := &GenericReporter{producer: &gtxCapturingProducer{}, category: gtxCategory}

		// an interface value with a type that isn't registered with gob can't be encoded.
		type unregistered struct{ V int }
		_, err := ReporterSendEvent(reporter, ctx, gtxEventType, struct{ Value any }{Value: unregistered{V: 1}})
		require.ErrorContains(t, err, "failed to encode payload")
	})
}

func TestGtxDiscardEventError(t *testing.T) {
	inner := errors.New("gtx inner")

	t.Run("wrapped error", func(t *testing.T) {
		err := NewDiscardEventError(inner)
		require.ErrorContains(t, err, "discarding requested due to: gtx inner")
		require.ErrorIs(t, err, inner)
		require.ErrorIs(t, err, errDiscardEvent)
	})

	t.Run("formatted error", func(t *testing.T) {
		err := NewDiscardEventErrorf("value %d", 7)
		require.ErrorContains(t, err, "discarding requested due to: value 7")
		require.ErrorIs(t, err, errDiscardEvent)
	})

	t.Run("other errors are not discard errors", func(t *testing.T) {
		assert.False(t, errors.Is(inner, errDiscardEvent))
	})
}

func TestGtxNoopCollector(t *testing.T) {
	c := NewNoopCollector()
	assert.NotPanics(t, func() {
		c.RecordError("category", "group")
		c.RecordDiscard("category", "group")
	})
}

func TestGtxStreamIDAndMode(t *testing.T) {
	assert.Equal(t, "events:cat:type", getStreamID("cat", "type"))
	assert.Equal(t, Mode("redis"), ModeRedis)
	assert.Equal(t, Mode("inmemory"), ModeInMemory)
}
