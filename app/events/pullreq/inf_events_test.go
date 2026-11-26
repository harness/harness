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
		require.Equal(t, "events:pullreq:branch-updated", sent.streamID)
		h.deliver(ctx, t, sent)
		require.Equal(t, want, got)

		before := h.producer.count()
		// a nil payload is never sent.
		h.reporter.BranchUpdated(ctx, nil)
		require.Equal(t, before, h.producer.count())

		// failures of the producer are handled.
		h.producer.err = errors.New("send failed")
		h.reporter.BranchUpdated(ctx, want)
		require.Equal(t, before, h.producer.count())
	})

	t.Run("TargetBranchChanged", func(t *testing.T) {
		h := infNewHarness(t)
		want := &TargetBranchChangedPayload{}
		infFill(reflect.ValueOf(want).Elem())
		var got *TargetBranchChangedPayload
		h.launch(ctx, t, func(r *Reader) error {
			return r.RegisterTargetBranchChanged(func(_ context.Context, e *events.Event[*TargetBranchChangedPayload]) error {
				got = e.Payload
				return nil
			})
		})

		h.reporter.TargetBranchChanged(ctx, want)
		sent := h.producer.last(t)
		require.Equal(t, "events:pullreq:target-branch-changed", sent.streamID)
		h.deliver(ctx, t, sent)
		require.Equal(t, want, got)

		before := h.producer.count()
		// a nil payload is never sent.
		h.reporter.TargetBranchChanged(ctx, nil)
		require.Equal(t, before, h.producer.count())

		// failures of the producer are handled.
		h.producer.err = errors.New("send failed")
		h.reporter.TargetBranchChanged(ctx, want)
		require.Equal(t, before, h.producer.count())
	})

	t.Run("CommentStatusUpdated", func(t *testing.T) {
		h := infNewHarness(t)
		want := &CommentStatusUpdatedPayload{}
		infFill(reflect.ValueOf(want).Elem())
		var got *CommentStatusUpdatedPayload
		h.launch(ctx, t, func(r *Reader) error {
			return r.RegisterCommentStatusUpdated(func(_ context.Context, e *events.Event[*CommentStatusUpdatedPayload]) error {
				got = e.Payload
				return nil
			})
		})

		h.reporter.CommentStatusUpdated(ctx, want)
		sent := h.producer.last(t)
		require.Equal(t, "events:pullreq:comment-status-updated", sent.streamID)
		h.deliver(ctx, t, sent)
		require.Equal(t, want, got)

		before := h.producer.count()
		// a nil payload is never sent.
		h.reporter.CommentStatusUpdated(ctx, nil)
		require.Equal(t, before, h.producer.count())

		// failures of the producer are handled.
		h.producer.err = errors.New("send failed")
		h.reporter.CommentStatusUpdated(ctx, want)
		require.Equal(t, before, h.producer.count())
	})

	t.Run("CommentUpdated", func(t *testing.T) {
		h := infNewHarness(t)
		want := &CommentUpdatedPayload{}
		infFill(reflect.ValueOf(want).Elem())
		var got *CommentUpdatedPayload
		h.launch(ctx, t, func(r *Reader) error {
			return r.RegisterCommentUpdated(func(_ context.Context, e *events.Event[*CommentUpdatedPayload]) error {
				got = e.Payload
				return nil
			})
		})

		h.reporter.CommentUpdated(ctx, want)
		sent := h.producer.last(t)
		require.Equal(t, "events:pullreq:comment-updated", sent.streamID)
		h.deliver(ctx, t, sent)
		require.Equal(t, want, got)

		before := h.producer.count()
		// a nil payload is never sent.
		h.reporter.CommentUpdated(ctx, nil)
		require.Equal(t, before, h.producer.count())

		// failures of the producer are handled.
		h.producer.err = errors.New("send failed")
		h.reporter.CommentUpdated(ctx, want)
		require.Equal(t, before, h.producer.count())
	})

	t.Run("CommentCreated", func(t *testing.T) {
		h := infNewHarness(t)
		want := &CommentCreatedPayload{}
		infFill(reflect.ValueOf(want).Elem())
		var got *CommentCreatedPayload
		h.launch(ctx, t, func(r *Reader) error {
			return r.RegisterCommentCreated(func(_ context.Context, e *events.Event[*CommentCreatedPayload]) error {
				got = e.Payload
				return nil
			})
		})

		h.reporter.CommentCreated(ctx, want)
		sent := h.producer.last(t)
		require.Equal(t, "events:pullreq:comment-created", sent.streamID)
		h.deliver(ctx, t, sent)
		require.Equal(t, want, got)

		before := h.producer.count()
		// a nil payload is never sent.
		h.reporter.CommentCreated(ctx, nil)
		require.Equal(t, before, h.producer.count())

		// failures of the producer are handled.
		h.producer.err = errors.New("send failed")
		h.reporter.CommentCreated(ctx, want)
		require.Equal(t, before, h.producer.count())
	})

	t.Run("LabelAssigned", func(t *testing.T) {
		h := infNewHarness(t)
		want := &LabelAssignedPayload{}
		infFill(reflect.ValueOf(want).Elem())
		var got *LabelAssignedPayload
		h.launch(ctx, t, func(r *Reader) error {
			return r.RegisterLabelAssigned(func(_ context.Context, e *events.Event[*LabelAssignedPayload]) error {
				got = e.Payload
				return nil
			})
		})

		h.reporter.LabelAssigned(ctx, want)
		sent := h.producer.last(t)
		require.Equal(t, "events:pullreq:label-assigned", sent.streamID)
		h.deliver(ctx, t, sent)
		require.Equal(t, want, got)

		before := h.producer.count()
		// a nil payload is never sent.
		h.reporter.LabelAssigned(ctx, nil)
		require.Equal(t, before, h.producer.count())

		// failures of the producer are handled.
		h.producer.err = errors.New("send failed")
		h.reporter.LabelAssigned(ctx, want)
		require.Equal(t, before, h.producer.count())
	})

	t.Run("ReviewSubmitted", func(t *testing.T) {
		h := infNewHarness(t)
		want := &ReviewSubmittedPayload{}
		infFill(reflect.ValueOf(want).Elem())
		var got *ReviewSubmittedPayload
		h.launch(ctx, t, func(r *Reader) error {
			return r.RegisterReviewSubmitted(func(_ context.Context, e *events.Event[*ReviewSubmittedPayload]) error {
				got = e.Payload
				return nil
			})
		})

		h.reporter.ReviewSubmitted(ctx, want)
		sent := h.producer.last(t)
		require.Equal(t, "events:pullreq:review-submitted", sent.streamID)
		h.deliver(ctx, t, sent)
		require.Equal(t, want, got)

		before := h.producer.count()
		// a nil payload is never sent.
		h.reporter.ReviewSubmitted(ctx, nil)
		require.Equal(t, before, h.producer.count())

		// failures of the producer are handled.
		h.producer.err = errors.New("send failed")
		h.reporter.ReviewSubmitted(ctx, want)
		require.Equal(t, before, h.producer.count())
	})

	t.Run("ReviewerAdded", func(t *testing.T) {
		h := infNewHarness(t)
		want := &ReviewerAddedPayload{}
		infFill(reflect.ValueOf(want).Elem())
		var got *ReviewerAddedPayload
		h.launch(ctx, t, func(r *Reader) error {
			return r.RegisterReviewerAdded(func(_ context.Context, e *events.Event[*ReviewerAddedPayload]) error {
				got = e.Payload
				return nil
			})
		})

		h.reporter.ReviewerAdded(ctx, want)
		sent := h.producer.last(t)
		require.Equal(t, "events:pullreq:reviewer-added", sent.streamID)
		h.deliver(ctx, t, sent)
		require.Equal(t, want, got)

		before := h.producer.count()
		// a nil payload is never sent.
		h.reporter.ReviewerAdded(ctx, nil)
		require.Equal(t, before, h.producer.count())

		// failures of the producer are handled.
		h.producer.err = errors.New("send failed")
		h.reporter.ReviewerAdded(ctx, want)
		require.Equal(t, before, h.producer.count())
	})

	t.Run("ReviewerRemoved", func(t *testing.T) {
		h := infNewHarness(t)
		want := &ReviewerRemovedPayload{}
		infFill(reflect.ValueOf(want).Elem())
		var got *ReviewerRemovedPayload
		h.launch(ctx, t, func(r *Reader) error {
			return r.RegisterReviewerRemoved(func(_ context.Context, e *events.Event[*ReviewerRemovedPayload]) error {
				got = e.Payload
				return nil
			})
		})

		h.reporter.ReviewerRemoved(ctx, want)
		sent := h.producer.last(t)
		require.Equal(t, "events:pullreq:reviewer-removed", sent.streamID)
		h.deliver(ctx, t, sent)
		require.Equal(t, want, got)

		before := h.producer.count()
		// a nil payload is never sent.
		h.reporter.ReviewerRemoved(ctx, nil)
		require.Equal(t, before, h.producer.count())

		// failures of the producer are handled.
		h.producer.err = errors.New("send failed")
		h.reporter.ReviewerRemoved(ctx, want)
		require.Equal(t, before, h.producer.count())
	})

	t.Run("UserGroupReviewerAdded", func(t *testing.T) {
		h := infNewHarness(t)
		want := &UserGroupReviewerAddedPayload{}
		infFill(reflect.ValueOf(want).Elem())
		var got *UserGroupReviewerAddedPayload
		h.launch(ctx, t, func(r *Reader) error {
			return r.RegisterUserGroupReviewerAdded(func(
				_ context.Context, e *events.Event[*UserGroupReviewerAddedPayload],
			) error {
				got = e.Payload
				return nil
			})
		})

		h.reporter.UserGroupReviewerAdded(ctx, want)
		sent := h.producer.last(t)
		require.Equal(t, "events:pullreq:usergroup-reviewer-added", sent.streamID)
		h.deliver(ctx, t, sent)
		require.Equal(t, want, got)

		before := h.producer.count()
		// a nil payload is never sent.
		h.reporter.UserGroupReviewerAdded(ctx, nil)
		require.Equal(t, before, h.producer.count())

		// failures of the producer are handled.
		h.producer.err = errors.New("send failed")
		h.reporter.UserGroupReviewerAdded(ctx, want)
		require.Equal(t, before, h.producer.count())
	})

	t.Run("UserGroupReviewerRemoved", func(t *testing.T) {
		h := infNewHarness(t)
		want := &UserGroupReviewerRemovedPayload{}
		infFill(reflect.ValueOf(want).Elem())
		var got *UserGroupReviewerRemovedPayload
		h.launch(ctx, t, func(r *Reader) error {
			return r.RegisterUserGroupReviewerRemoved(func(
				_ context.Context, e *events.Event[*UserGroupReviewerRemovedPayload],
			) error {
				got = e.Payload
				return nil
			})
		})

		h.reporter.UserGroupReviewerRemoved(ctx, want)
		sent := h.producer.last(t)
		require.Equal(t, "events:pullreq:usergroup-reviewer-removed", sent.streamID)
		h.deliver(ctx, t, sent)
		require.Equal(t, want, got)

		before := h.producer.count()
		// a nil payload is never sent.
		h.reporter.UserGroupReviewerRemoved(ctx, nil)
		require.Equal(t, before, h.producer.count())

		// failures of the producer are handled.
		h.producer.err = errors.New("send failed")
		h.reporter.UserGroupReviewerRemoved(ctx, want)
		require.Equal(t, before, h.producer.count())
	})

	t.Run("Created", func(t *testing.T) {
		h := infNewHarness(t)
		want := &CreatedPayload{}
		infFill(reflect.ValueOf(want).Elem())
		var got *CreatedPayload
		h.launch(ctx, t, func(r *Reader) error {
			return r.RegisterCreated(func(_ context.Context, e *events.Event[*CreatedPayload]) error {
				got = e.Payload
				return nil
			})
		})

		h.reporter.Created(ctx, want)
		sent := h.producer.last(t)
		require.Equal(t, "events:pullreq:created", sent.streamID)
		h.deliver(ctx, t, sent)
		require.Equal(t, want, got)

		before := h.producer.count()
		// a nil payload is never sent.
		h.reporter.Created(ctx, nil)
		require.Equal(t, before, h.producer.count())

		// failures of the producer are handled.
		h.producer.err = errors.New("send failed")
		h.reporter.Created(ctx, want)
		require.Equal(t, before, h.producer.count())
	})

	t.Run("Closed", func(t *testing.T) {
		h := infNewHarness(t)
		want := &ClosedPayload{}
		infFill(reflect.ValueOf(want).Elem())
		var got *ClosedPayload
		h.launch(ctx, t, func(r *Reader) error {
			return r.RegisterClosed(func(_ context.Context, e *events.Event[*ClosedPayload]) error {
				got = e.Payload
				return nil
			})
		})

		h.reporter.Closed(ctx, want)
		sent := h.producer.last(t)
		require.Equal(t, "events:pullreq:closed", sent.streamID)
		h.deliver(ctx, t, sent)
		require.Equal(t, want, got)

		before := h.producer.count()
		// a nil payload is never sent.
		h.reporter.Closed(ctx, nil)
		require.Equal(t, before, h.producer.count())

		// failures of the producer are handled.
		h.producer.err = errors.New("send failed")
		h.reporter.Closed(ctx, want)
		require.Equal(t, before, h.producer.count())
	})

	t.Run("Reopened", func(t *testing.T) {
		h := infNewHarness(t)
		want := &ReopenedPayload{}
		infFill(reflect.ValueOf(want).Elem())
		var got *ReopenedPayload
		h.launch(ctx, t, func(r *Reader) error {
			return r.RegisterReopened(func(_ context.Context, e *events.Event[*ReopenedPayload]) error {
				got = e.Payload
				return nil
			})
		})

		h.reporter.Reopened(ctx, want)
		sent := h.producer.last(t)
		require.Equal(t, "events:pullreq:reopened", sent.streamID)
		h.deliver(ctx, t, sent)
		require.Equal(t, want, got)

		before := h.producer.count()
		// a nil payload is never sent.
		h.reporter.Reopened(ctx, nil)
		require.Equal(t, before, h.producer.count())

		// failures of the producer are handled.
		h.producer.err = errors.New("send failed")
		h.reporter.Reopened(ctx, want)
		require.Equal(t, before, h.producer.count())
	})

	t.Run("Merged", func(t *testing.T) {
		h := infNewHarness(t)
		want := &MergedPayload{}
		infFill(reflect.ValueOf(want).Elem())
		var got *MergedPayload
		h.launch(ctx, t, func(r *Reader) error {
			return r.RegisterMerged(func(_ context.Context, e *events.Event[*MergedPayload]) error {
				got = e.Payload
				return nil
			})
		})

		h.reporter.Merged(ctx, want)
		sent := h.producer.last(t)
		require.Equal(t, "events:pullreq:merged", sent.streamID)
		h.deliver(ctx, t, sent)
		require.Equal(t, want, got)

		before := h.producer.count()
		// a nil payload is never sent.
		h.reporter.Merged(ctx, nil)
		require.Equal(t, before, h.producer.count())

		// failures of the producer are handled.
		h.producer.err = errors.New("send failed")
		h.reporter.Merged(ctx, want)
		require.Equal(t, before, h.producer.count())
	})

	t.Run("Updated", func(t *testing.T) {
		h := infNewHarness(t)
		want := &UpdatedPayload{}
		infFill(reflect.ValueOf(want).Elem())
		var got *UpdatedPayload
		h.launch(ctx, t, func(r *Reader) error {
			return r.RegisterUpdated(func(_ context.Context, e *events.Event[*UpdatedPayload]) error {
				got = e.Payload
				return nil
			})
		})

		h.reporter.Updated(ctx, want)
		sent := h.producer.last(t)
		require.Equal(t, "events:pullreq:updated", sent.streamID)
		h.deliver(ctx, t, sent)
		require.Equal(t, want, got)

		before := h.producer.count()
		// a nil payload is never sent.
		h.reporter.Updated(ctx, nil)
		require.Equal(t, before, h.producer.count())

		// failures of the producer are handled.
		h.producer.err = errors.New("send failed")
		h.reporter.Updated(ctx, want)
		require.Equal(t, before, h.producer.count())
	})

	t.Run("MergeCheckSucceeded", func(t *testing.T) {
		h := infNewHarness(t)
		want := &MergeCheckSucceededPayload{}
		infFill(reflect.ValueOf(want).Elem())
		var got *MergeCheckSucceededPayload
		h.launch(ctx, t, func(r *Reader) error {
			return r.RegisterMergeCheckSucceeded(func(_ context.Context, e *events.Event[*MergeCheckSucceededPayload]) error {
				got = e.Payload
				return nil
			})
		})

		h.reporter.MergeCheckSucceeded(ctx, want)
		sent := h.producer.last(t)
		require.Equal(t, "events:pullreq:merge-check-succeeded", sent.streamID)
		h.deliver(ctx, t, sent)
		require.Equal(t, want, got)

		before := h.producer.count()
		// a nil payload is never sent.
		h.reporter.MergeCheckSucceeded(ctx, nil)
		require.Equal(t, before, h.producer.count())

		// failures of the producer are handled.
		h.producer.err = errors.New("send failed")
		h.reporter.MergeCheckSucceeded(ctx, want)
		require.Equal(t, before, h.producer.count())
	})
}

func TestInfEvents_ReaderFactoryInvalidSystem(t *testing.T) {
	_, err := NewReaderFactory(nil)
	require.Error(t, err)
	_, err = NewReporter(nil)
	require.Error(t, err)
}
