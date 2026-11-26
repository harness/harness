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

package asyncprocessing

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/stream"
	gitnesstypes "github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

var errW2rBoom = errors.New("boom")

type w2rProducer struct {
	mu   sync.Mutex
	sent int
	err  error
}

func (p *w2rProducer) Send(context.Context, string, map[string]any) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.err != nil {
		return "", p.err
	}
	p.sent++
	return "1-0", nil
}

type w2rConsumer struct {
	mu       sync.Mutex
	handlers map[string]stream.HandlerFunc
}

func (c *w2rConsumer) Register(streamID string, fn stream.HandlerFunc, _ ...stream.HandlerOption) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[streamID] = fn
	return nil
}
func (c *w2rConsumer) Configure(...stream.ConsumerOption) {}
func (c *w2rConsumer) Start(context.Context) error        { return nil }
func (c *w2rConsumer) Errors() <-chan error               { ch := make(chan error); close(ch); return ch }
func (c *w2rConsumer) Infos() <-chan string               { ch := make(chan string); close(ch); return ch }

type w2rTx struct{}

func (w2rTx) WithTx(ctx context.Context, fn func(context.Context) error, _ ...any) error {
	return fn(ctx)
}

type w2rTasks struct {
	store.TaskRepository
	status      types.TaskStatus
	upsertErr   error
	lockErr     error
	updateErr   error
	runAgainErr error
	upserted    []string
	runAgain    []string
	updated     []types.TaskStatus
}

func (t *w2rTasks) UpsertTask(_ context.Context, task *types.Task) error {
	t.upserted = append(t.upserted, task.Key)
	return t.upsertErr
}

func (t *w2rTasks) LockForUpdate(context.Context, *types.Task) (types.TaskStatus, error) {
	return t.status, t.lockErr
}

func (t *w2rTasks) SetRunAgain(_ context.Context, key string, _ bool) error {
	t.runAgain = append(t.runAgain, key)
	return t.runAgainErr
}

func (t *w2rTasks) UpdateStatus(_ context.Context, _ string, status types.TaskStatus) error {
	t.updated = append(t.updated, status)
	return t.updateErr
}

type w2rSources struct {
	store.TaskSourceRepository
	err     error
	sources []types.SourceRef
}

func (s *w2rSources) InsertSource(_ context.Context, _ string, src types.SourceRef) error {
	s.sources = append(s.sources, src)
	return s.err
}

type w2rTaskEvents struct {
	store.TaskEventRepository
	err    error
	events []string
}

func (e *w2rTaskEvents) LogTaskEvent(_ context.Context, _ string, event string, _ []byte) error {
	e.events = append(e.events, event)
	return e.err
}

type w2rEnv struct {
	reporter *Reporter
	producer *w2rProducer
	consumer *w2rConsumer
	tasks    *w2rTasks
	sources  *w2rSources
	logs     *w2rTaskEvents
	ctx      context.Context
}

func w2rNewEnv(t *testing.T) *w2rEnv {
	t.Helper()
	producer := &w2rProducer{}
	consumer := &w2rConsumer{handlers: map[string]stream.HandlerFunc{}}
	system, err := events.NewSystem(
		func(string, string) (events.StreamConsumer, error) { return consumer, nil },
		producer, events.NewNoopCollector())
	require.NoError(t, err)

	tasks, sources, logs := &w2rTasks{status: types.TaskStatusPending}, &w2rSources{}, &w2rTaskEvents{}
	reporter, err := NewReporter(w2rTx{}, system, tasks, sources, logs)
	require.NoError(t, err)

	ctx := request.WithAuthSession(context.Background(), &auth.Session{Principal: gitnesstypes.Principal{ID: 42}})
	return &w2rEnv{reporter, producer, consumer, tasks, sources, logs, ctx}
}

func TestW2rReporterEnqueuesIndexTasks(t *testing.T) {
	env := w2rNewEnv(t)

	env.reporter.BuildRegistryIndex(env.ctx, 7, []types.SourceRef{{Type: types.SourceTypeRegistry, ID: 1}})
	env.reporter.BuildPackageIndex(env.ctx, 7, "img")
	env.reporter.BuildPackageMetadata(env.ctx, 7, "img", "1.0")

	require.Equal(t, []string{"registry_7", "package_7_img", "package_7_img_1.0_metadata"}, env.tasks.upserted)
	require.Equal(t, 3, env.producer.sent, "every pending task is announced on the event stream")
	require.Equal(t, []string{"enqueued", "enqueued", "enqueued"}, env.logs.events)
	require.Len(t, env.sources.sources, 4, "the registry source is added to the given ones")
	require.Equal(t, []types.TaskStatus{
		types.TaskStatusPending, types.TaskStatusPending, types.TaskStatusPending,
	}, env.tasks.updated)
}

func TestW2rReporterMergesIntoRunningTask(t *testing.T) {
	env := w2rNewEnv(t)
	env.tasks.status = types.TaskStatusProcessing

	env.reporter.BuildPackageIndexWithPrincipal(env.ctx, 7, "img", 42)

	require.Equal(t, []string{"package_7_img"}, env.tasks.runAgain)
	require.Equal(t, []string{"merged"}, env.logs.events)
	require.Zero(t, env.producer.sent, "a running task is re-run instead of enqueued again")
	require.Empty(t, env.tasks.updated)

	// A failing merge log is only logged.
	env.logs.err = errW2rBoom
	env.reporter.BuildRegistryIndexWithPrincipal(env.ctx, 7, nil, 42)
	require.Zero(t, env.producer.sent)
}

func TestW2rReporterUpsertFailures(t *testing.T) {
	tests := map[string]func(*w2rEnv){
		"upsert":     func(e *w2rEnv) { e.tasks.upsertErr = errW2rBoom },
		"lock":       func(e *w2rEnv) { e.tasks.lockErr = errW2rBoom },
		"source":     func(e *w2rEnv) { e.sources.err = errW2rBoom },
		"update":     func(e *w2rEnv) { e.tasks.updateErr = errW2rBoom },
		"log":        func(e *w2rEnv) { e.logs.err = errW2rBoom },
		"run again":  func(e *w2rEnv) { e.tasks.status = types.TaskStatusProcessing; e.tasks.runAgainErr = errW2rBoom },
		"event send": func(e *w2rEnv) { e.producer.err = errW2rBoom },
	}
	for name, setup := range tests {
		env := w2rNewEnv(t)
		setup(env)

		err := env.reporter.UpsertAndSendEvent(env.ctx, &types.Task{Key: "k"}, []types.SourceRef{{ID: 1}})
		require.Error(t, err, name)
	}
}

func TestW2rReaderRegistersAsyncTaskHandler(t *testing.T) {
	producer := &w2rProducer{}
	consumer := &w2rConsumer{handlers: map[string]stream.HandlerFunc{}}
	system, err := events.NewSystem(
		func(string, string) (events.StreamConsumer, error) { return consumer, nil },
		producer, events.NewNoopCollector())
	require.NoError(t, err)

	factory, err := NewReaderFactory(system)
	require.NoError(t, err)

	var got *events.Event[*ExecuteAsyncTaskPayload]
	canceler, err := factory.Launch(context.Background(), "grp", "reader", func(r *Reader) error {
		r.Configure(events.WithConcurrency(1))
		return r.RegisterExecuteAsyncTask(func(_ context.Context, e *events.Event[*ExecuteAsyncTaskPayload]) error {
			got = e
			return nil
		})
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = canceler.Cancel() })
	require.Len(t, consumer.handlers, 1)
	require.Nil(t, got, "no event was delivered yet")
}
