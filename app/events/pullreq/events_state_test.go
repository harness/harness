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
	"testing"
	"time"

	"github.com/harness/gitness/events"

	"github.com/stretchr/testify/require"
)

func newInMemorySystem(t *testing.T) *events.System {
	t.Helper()

	system, err := events.ProvideSystem(events.Config{
		Mode:            events.ModeInMemory,
		MaxStreamLength: 100,
	}, nil, events.NewNoopCollector())
	require.NoError(t, err)

	return system
}

// sentinelPullReqID marks readiness probes. Handlers forward probes to a ready
// channel instead of treating them as real events.
const sentinelPullReqID int64 = -1

// waitUntilReady repeatedly invokes send until ready receives a value.
//
// The in-memory consumer subscribes each stream to the broker from a goroutine
// started by Launch, and the broker drops messages for streams without a
// subscribed group. Probing avoids racing that subscription.
func waitUntilReady(t *testing.T, ready <-chan struct{}, send func()) {
	t.Helper()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.After(5 * time.Second)

	send()
	for {
		select {
		case <-ready:
			// drain probes that were already in flight.
			time.Sleep(50 * time.Millisecond)
			for len(ready) > 0 {
				<-ready
			}
			return
		case <-ticker.C:
			send()
		case <-timeout:
			t.Fatal("timed out waiting for event reader to subscribe")
		}
	}
}

type draftHarness struct {
	reporter *Reporter
	ready    chan struct{}
	received chan *events.Event[*DraftChangedPayload]
}

// newDraftHarness launches a reader with a DraftChanged handler (plus any
// extra registrations) and blocks until the draft stream is subscribed.
func newDraftHarness(t *testing.T, ctx context.Context, extra func(r *Reader) error) *draftHarness {
	t.Helper()

	system := newInMemorySystem(t)

	readerFactory, err := NewReaderFactory(system)
	require.NoError(t, err)

	h := &draftHarness{
		ready:    make(chan struct{}, 100),
		received: make(chan *events.Event[*DraftChangedPayload], 10),
	}

	_, err = readerFactory.Launch(ctx, "test-group", "test-reader", func(r *Reader) error {
		if err := r.RegisterDraftChanged(
			func(_ context.Context, e *events.Event[*DraftChangedPayload]) error {
				if e.Payload != nil && e.Payload.PullReqID == sentinelPullReqID {
					h.ready <- struct{}{}
					return nil
				}
				h.received <- e
				return nil
			}); err != nil {
			return err
		}
		if extra != nil {
			return extra(r)
		}
		return nil
	})
	require.NoError(t, err)

	h.reporter, err = NewReporter(system)
	require.NoError(t, err)

	waitUntilReady(t, h.ready, func() {
		h.reporter.DraftChanged(ctx, &DraftChangedPayload{Base: Base{PullReqID: sentinelPullReqID}})
	})

	return h
}

func (h *draftHarness) next(t *testing.T) *events.Event[*DraftChangedPayload] {
	t.Helper()

	select {
	case e := <-h.received:
		return e
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for draft changed event")
		return nil
	}
}

// TestDraftChanged_RoundTrip verifies that a DraftChanged event sent through the
// Reporter is delivered to a handler registered via RegisterDraftChanged with
// the payload intact.
func TestDraftChanged_RoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		isDraft bool
	}{
		{name: "marked as draft", isDraft: true},
		{name: "marked ready for review", isDraft: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			h := newDraftHarness(t, ctx, nil)

			sourceRepoID := int64(22)
			sent := &DraftChangedPayload{
				Base: Base{
					PullReqID:    11,
					SourceRepoID: &sourceRepoID,
					TargetRepoID: 33,
					PrincipalID:  44,
					Number:       55,
				},
				IsDraft: tt.isDraft,
			}
			h.reporter.DraftChanged(ctx, sent)

			e := h.next(t)
			require.NotEmpty(t, e.ID)
			require.Equal(t, sent, e.Payload)
		})
	}
}

// TestDraftChanged_NilPayload verifies a nil payload is dropped without
// publishing anything (and without panicking).
func TestDraftChanged_NilPayload(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h := newDraftHarness(t, ctx, nil)

	h.reporter.DraftChanged(ctx, nil)
	// a follow-up event: if the nil payload had been published, it would be delivered first.
	h.reporter.DraftChanged(ctx, &DraftChangedPayload{Base: Base{PullReqID: 7}, IsDraft: true})

	e := h.next(t)
	require.NotNil(t, e.Payload, "nil payload must not produce an event")
	require.Equal(t, int64(7), e.Payload.PullReqID)
}

// TestDraftChanged_DoesNotTriggerOtherHandlers guards against the draft event
// type colliding with the closed/reopened event types on the same reader.
func TestDraftChanged_DoesNotTriggerOtherHandlers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	closedReady := make(chan struct{}, 100)
	reopenedReady := make(chan struct{}, 100)
	other := make(chan events.EventType, 10)

	h := newDraftHarness(t, ctx, func(r *Reader) error {
		if err := r.RegisterClosed(
			func(_ context.Context, e *events.Event[*ClosedPayload]) error {
				if e.Payload.PullReqID == sentinelPullReqID {
					closedReady <- struct{}{}
					return nil
				}
				other <- ClosedEvent
				return nil
			}); err != nil {
			return err
		}
		return r.RegisterReopened(
			func(_ context.Context, e *events.Event[*ReopenedPayload]) error {
				if e.Payload.PullReqID == sentinelPullReqID {
					reopenedReady <- struct{}{}
					return nil
				}
				other <- ReopenedEvent
				return nil
			})
	})

	// make sure the other streams are subscribed too, otherwise a misrouted event could be silently dropped.
	waitUntilReady(t, closedReady, func() {
		h.reporter.Closed(ctx, &ClosedPayload{Base: Base{PullReqID: sentinelPullReqID}})
	})
	waitUntilReady(t, reopenedReady, func() {
		h.reporter.Reopened(ctx, &ReopenedPayload{Base: Base{PullReqID: sentinelPullReqID}})
	})

	h.reporter.DraftChanged(ctx, &DraftChangedPayload{Base: Base{PullReqID: 1}, IsDraft: true})

	e := h.next(t)
	require.Equal(t, int64(1), e.Payload.PullReqID)

	select {
	case et := <-other:
		t.Fatalf("draft changed event unexpectedly delivered to %s handler", et)
	case <-time.After(200 * time.Millisecond):
	}
}
