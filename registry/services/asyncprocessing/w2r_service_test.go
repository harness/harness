//  Copyright 2023 Harness, Inc.
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
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/harness/gitness/app/services/locker"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/lock"
	"github.com/harness/gitness/registry/app/api/interfaces"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	eventsasync "github.com/harness/gitness/registry/app/events/asyncprocessing"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/stream"

	"github.com/stretchr/testify/assert"
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

func (p *w2rProducer) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.sent
}

type w2rConsumer struct{}

func (w2rConsumer) Register(string, stream.HandlerFunc, ...stream.HandlerOption) error { return nil }
func (w2rConsumer) Configure(...stream.ConsumerOption)                                 {}
func (w2rConsumer) Start(context.Context) error                                        { return nil }
func (w2rConsumer) Errors() <-chan error {
	ch := make(chan error)
	close(ch)
	return ch
}
func (w2rConsumer) Infos() <-chan string {
	ch := make(chan string)
	close(ch)
	return ch
}

type w2rTx struct{}

func (w2rTx) WithTx(ctx context.Context, fn func(context.Context) error, _ ...any) error {
	return fn(ctx)
}

type w2rTasks struct {
	store.TaskRepository
	task        *types.Task
	findErr     error
	lockErr     error
	updateErr   error
	runAgainErr error
	completeErr error
	runAgain    bool
	completed   []types.TaskStatus
}

func (t *w2rTasks) Find(context.Context, string) (*types.Task, error) { return t.task, t.findErr }

func (t *w2rTasks) UpsertTask(context.Context, *types.Task) error { return nil }

func (t *w2rTasks) LockForUpdate(context.Context, *types.Task) (types.TaskStatus, error) {
	return types.TaskStatusPending, t.lockErr
}

func (t *w2rTasks) UpdateStatus(context.Context, string, types.TaskStatus) error { return t.updateErr }

func (t *w2rTasks) SetRunAgain(context.Context, string, bool) error { return t.runAgainErr }

func (t *w2rTasks) CompleteTask(_ context.Context, _ string, status types.TaskStatus) (bool, error) {
	t.completed = append(t.completed, status)
	return t.runAgain, t.completeErr
}

type w2rSources struct {
	store.TaskSourceRepository
	claimErr  error
	statusErr error
	statuses  []types.TaskStatus
	messages  []string
}

func (s *w2rSources) InsertSource(context.Context, string, types.SourceRef) error { return nil }

func (s *w2rSources) ClaimSources(context.Context, string, string) error { return s.claimErr }

func (s *w2rSources) UpdateSourceStatus(_ context.Context, _ string, st types.TaskStatus, msg string) error {
	s.statuses = append(s.statuses, st)
	s.messages = append(s.messages, msg)
	return s.statusErr
}

type w2rTaskEvents struct {
	store.TaskEventRepository
	err    error
	events []string
}

func (e *w2rTaskEvents) LogTaskEvent(_ context.Context, _ string, ev string, _ []byte) error {
	e.events = append(e.events, ev)
	return e.err
}

type w2rRegistries struct {
	store.RegistryRepository
	registry *types.Registry
	getErr   error
	ids      []int64
	idsErr   error
}

func (r *w2rRegistries) Get(context.Context, int64, ...types.QueryOption) (*types.Registry, error) {
	return r.registry, r.getErr
}

func (r *w2rRegistries) FetchRegistriesIDByUpstreamProxyID(
	context.Context, string, int64, ...types.QueryOption,
) ([]int64, error) {
	return r.ids, r.idsErr
}

type w2rWrapper struct {
	interfaces.PackageWrapper
	err   error
	calls []string
}

func (w *w2rWrapper) BuildRegistryIndexAsync(context.Context, types.BuildRegistryIndexTaskPayload) error {
	w.calls = append(w.calls, "registry")
	return w.err
}

func (w *w2rWrapper) BuildPackageIndexAsync(context.Context, types.BuildPackageIndexTaskPayload) error {
	w.calls = append(w.calls, "index")
	return w.err
}

func (w *w2rWrapper) BuildPackageMetadataAsync(context.Context, types.BuildPackageMetadataTaskPayload) error {
	w.calls = append(w.calls, "metadata")
	return w.err
}

type w2rRpm struct {
	err   error
	calls int
}

func (r *w2rRpm) BuildRegistryFiles(context.Context, types.Registry, int64) error {
	r.calls++
	return r.err
}

type w2rGo struct {
	err   error
	calls []string
}

func (g *w2rGo) UpdatePackageIndex(context.Context, int64, int64, int64, string) error {
	g.calls = append(g.calls, "index")
	return g.err
}

func (g *w2rGo) UpdatePackageMetadata(context.Context, int64, int64, string, string) error {
	g.calls = append(g.calls, "metadata")
	return g.err
}

type w2rEnv struct {
	svc      *Service
	producer *w2rProducer
	tasks    *w2rTasks
	sources  *w2rSources
	logs     *w2rTaskEvents
	regs     *w2rRegistries
	wrapper  *w2rWrapper
	rpm      *w2rRpm
	goHelper *w2rGo
}

func w2rNewEnv(t *testing.T) *w2rEnv {
	t.Helper()
	producer := &w2rProducer{}
	system, err := events.NewSystem(
		func(string, string) (events.StreamConsumer, error) { return w2rConsumer{}, nil },
		producer, events.NewNoopCollector())
	require.NoError(t, err)

	env := &w2rEnv{
		producer: producer,
		tasks:    &w2rTasks{},
		sources:  &w2rSources{},
		logs:     &w2rTaskEvents{},
		regs:     &w2rRegistries{registry: &types.Registry{ID: 5, RootParentID: 1}},
		wrapper:  &w2rWrapper{},
		rpm:      &w2rRpm{},
		goHelper: &w2rGo{},
	}
	reporter, err := eventsasync.NewReporter(w2rTx{}, system, env.tasks, env.sources, env.logs)
	require.NoError(t, err)
	inner, err := events.NewReporter(system, eventsasync.RegistryAsyncProcessing)
	require.NoError(t, err)

	mtx := lock.NewInMemory(lock.Config{
		App: "t", Namespace: "n", Expiry: time.Second, Tries: 2, RetryDelay: time.Millisecond,
	})
	env.svc = &Service{
		tx:                      w2rTx{},
		rpmRegistryHelper:       env.rpm,
		gopackageRegistryHelper: env.goHelper,
		locker:                  locker.NewLocker(mtx),
		registryDao:             env.regs,
		taskRepository:          env.tasks,
		taskSourceRepository:    env.sources,
		taskEventRepository:     env.logs,
		innerReporter:           inner,
		postProcessingReporter:  reporter,
		packageWrapper:          env.wrapper,
		taskHandlers:            map[types.TaskKind]TaskHandler{},
	}
	return env
}

func w2rTask(t *testing.T, kind types.TaskKind, payload any) *types.Task {
	t.Helper()
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	return &types.Task{Key: "k", Kind: kind, Payload: raw}
}

func w2rEvent() *events.Event[*eventsasync.ExecuteAsyncTaskPayload] {
	return &events.Event[*eventsasync.ExecuteAsyncTaskPayload]{
		ID: "evt-1", Payload: &eventsasync.ExecuteAsyncTaskPayload{TaskKey: "k"},
	}
}

func TestW2rConfigPrepare(t *testing.T) {
	var nilCfg *Config
	require.Error(t, nilCfg.Prepare())
	require.Error(t, (&Config{Concurrency: 1}).Prepare())
	require.Error(t, (&Config{EventReaderName: "r"}).Prepare())
	require.Error(t, (&Config{EventReaderName: "r", Concurrency: 1, MaxRetries: -1}).Prepare())
	require.NoError(t, (&Config{EventReaderName: "r", Concurrency: 1}).Prepare())
}

func TestW2rNewService(t *testing.T) {
	system, err := events.NewSystem(
		func(string, string) (events.StreamConsumer, error) { return w2rConsumer{}, nil },
		&w2rProducer{}, events.NewNoopCollector())
	require.NoError(t, err)
	factory, err := eventsasync.NewReaderFactory(system)
	require.NoError(t, err)

	cfg := Config{EventReaderName: "r", Concurrency: 1, MaxRetries: 1}
	svc, err := NewService(context.Background(), w2rTx{}, nil, nil, nil, nil, factory, cfg,
		nil, nil, nil, nil, system, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, svc)

	_, err = NewService(context.Background(), w2rTx{}, nil, nil, nil, nil, factory, Config{},
		nil, nil, nil, nil, system, nil, nil)
	require.Error(t, err)
}

func TestW2rExecuteBuildRegistryIndex(t *testing.T) {
	ctx := context.Background()
	payload := types.BuildRegistryIndexTaskPayload{Key: "k", RegistryID: 5, PrincipalID: 9}

	t.Run("generic package type", func(t *testing.T) {
		env := w2rNewEnv(t)
		env.tasks.task = w2rTask(t, types.TaskKindBuildRegistryIndex, payload)
		require.NoError(t, env.svc.handleEventExecuteAsyncTask(ctx, w2rEvent()))
		assert.Equal(t, []string{"registry"}, env.wrapper.calls)
		assert.Equal(t, []types.TaskStatus{types.TaskStatusSuccess}, env.tasks.completed)
		assert.Equal(t, []string{"started", "success"}, env.logs.events)
	})

	t.Run("rpm builds files and fans out to virtual registries", func(t *testing.T) {
		env := w2rNewEnv(t)
		env.regs.registry.PackageType = artifact.PackageTypeRPM
		env.regs.ids = []int64{11, 12}
		env.tasks.task = w2rTask(t, types.TaskKindBuildRegistryIndex, payload)
		require.NoError(t, env.svc.handleEventExecuteAsyncTask(ctx, w2rEvent()))
		assert.Equal(t, 1, env.rpm.calls)
		assert.Equal(t, 2, env.producer.count(), "one event for each virtual registry")
	})

	t.Run("rpm failure is recorded", func(t *testing.T) {
		env := w2rNewEnv(t)
		env.regs.registry.PackageType = artifact.PackageTypeRPM
		env.regs.idsErr = errW2rBoom
		env.rpm.err = errW2rBoom
		env.tasks.task = w2rTask(t, types.TaskKindBuildRegistryIndex, payload)
		require.NoError(t, env.svc.handleEventExecuteAsyncTask(ctx, w2rEvent()))
		assert.Equal(t, []types.TaskStatus{types.TaskStatusFailure}, env.tasks.completed)
		assert.Contains(t, env.sources.messages[0], "RPM registry files")
		assert.Equal(t, []string{"started", "failed"}, env.logs.events)
	})

	t.Run("registry lookup and payload errors", func(t *testing.T) {
		env := w2rNewEnv(t)
		env.regs.getErr = errW2rBoom
		env.tasks.task = w2rTask(t, types.TaskKindBuildRegistryIndex, payload)
		require.NoError(t, env.svc.handleEventExecuteAsyncTask(ctx, w2rEvent()))
		assert.Contains(t, env.sources.messages[0], "get registry")

		env = w2rNewEnv(t)
		env.tasks.task = &types.Task{Key: "k", Kind: types.TaskKindBuildRegistryIndex, Payload: []byte("{")}
		require.NoError(t, env.svc.handleEventExecuteAsyncTask(ctx, w2rEvent()))
		assert.Contains(t, env.sources.messages[0], "unmarshal")
	})

	t.Run("wrapper failure", func(t *testing.T) {
		env := w2rNewEnv(t)
		env.wrapper.err = errW2rBoom
		env.tasks.task = w2rTask(t, types.TaskKindBuildRegistryIndex, payload)
		require.NoError(t, env.svc.handleEventExecuteAsyncTask(ctx, w2rEvent()))
		assert.Equal(t, []types.TaskStatus{types.TaskStatusFailure}, env.tasks.completed)
	})
}

func TestW2rExecuteBuildPackageIndexAndMetadata(t *testing.T) {
	ctx := context.Background()
	index := types.BuildPackageIndexTaskPayload{Key: "k", RegistryID: 5, Image: "img", PrincipalID: 9}
	meta := types.BuildPackageMetadataTaskPayload{Key: "k", RegistryID: 5, Image: "img", Version: "1", PrincipalID: 9}

	cases := []struct {
		name string
		kind types.TaskKind
		body any
		want string
	}{
		{"index", types.TaskKindBuildPackageIndex, index, "index"},
		{"metadata", types.TaskKindBuildPackageMetadata, meta, "metadata"},
	}
	for _, tc := range cases {
		t.Run(tc.name+" go", func(t *testing.T) {
			env := w2rNewEnv(t)
			env.regs.registry.PackageType = artifact.PackageTypeGO
			env.tasks.task = w2rTask(t, tc.kind, tc.body)
			require.NoError(t, env.svc.handleEventExecuteAsyncTask(ctx, w2rEvent()))
			assert.Equal(t, []string{tc.want}, env.goHelper.calls)
			assert.Equal(t, []types.TaskStatus{types.TaskStatusSuccess}, env.tasks.completed)
		})
		t.Run(tc.name+" go failure", func(t *testing.T) {
			env := w2rNewEnv(t)
			env.regs.registry.PackageType = artifact.PackageTypeGO
			env.goHelper.err = errW2rBoom
			env.tasks.task = w2rTask(t, tc.kind, tc.body)
			require.NoError(t, env.svc.handleEventExecuteAsyncTask(ctx, w2rEvent()))
			assert.Equal(t, []types.TaskStatus{types.TaskStatusFailure}, env.tasks.completed)
		})
		t.Run(tc.name+" wrapper", func(t *testing.T) {
			env := w2rNewEnv(t)
			env.tasks.task = w2rTask(t, tc.kind, tc.body)
			require.NoError(t, env.svc.handleEventExecuteAsyncTask(ctx, w2rEvent()))
			assert.Equal(t, []string{tc.want}, env.wrapper.calls)
		})
		t.Run(tc.name+" wrapper failure", func(t *testing.T) {
			env := w2rNewEnv(t)
			env.wrapper.err = errW2rBoom
			env.tasks.task = w2rTask(t, tc.kind, tc.body)
			require.NoError(t, env.svc.handleEventExecuteAsyncTask(ctx, w2rEvent()))
			assert.Equal(t, []types.TaskStatus{types.TaskStatusFailure}, env.tasks.completed)
		})
		t.Run(tc.name+" lookup and payload errors", func(t *testing.T) {
			env := w2rNewEnv(t)
			env.regs.getErr = errW2rBoom
			env.tasks.task = w2rTask(t, tc.kind, tc.body)
			require.NoError(t, env.svc.handleEventExecuteAsyncTask(ctx, w2rEvent()))
			assert.Contains(t, env.sources.messages[0], "get registry")

			env = w2rNewEnv(t)
			env.tasks.task = &types.Task{Key: "k", Kind: tc.kind, Payload: []byte("{")}
			require.NoError(t, env.svc.handleEventExecuteAsyncTask(ctx, w2rEvent()))
			assert.Contains(t, env.sources.messages[0], "unmarshal")
		})
	}
}

func TestW2rExecuteCustomAndUnknownKinds(t *testing.T) {
	ctx := context.Background()

	env := w2rNewEnv(t)
	env.tasks.task = &types.Task{Key: "k", Kind: "custom"}
	var handled string
	env.svc.RegisterTaskHandler("custom", func(_ context.Context, task *types.Task, eventID string) error {
		handled = task.Key + eventID
		return nil
	})
	require.NoError(t, env.svc.handleEventExecuteAsyncTask(ctx, w2rEvent()))
	assert.Equal(t, "kevt-1", handled)
	assert.Equal(t, []types.TaskStatus{types.TaskStatusSuccess}, env.tasks.completed)

	env = w2rNewEnv(t)
	env.tasks.task = &types.Task{Key: "k", Kind: "custom"}
	env.svc.RegisterTaskHandler("custom", func(context.Context, *types.Task, string) error { return errW2rBoom })
	require.NoError(t, env.svc.handleEventExecuteAsyncTask(ctx, w2rEvent()))
	assert.Contains(t, env.sources.messages[0], "custom")

	env = w2rNewEnv(t)
	env.tasks.task = &types.Task{Key: "k", Kind: "unknown"}
	require.NoError(t, env.svc.handleEventExecuteAsyncTask(ctx, w2rEvent()))
	assert.Contains(t, env.sources.messages[0], "unsupported task kind")
}

func TestW2rExecuteRunAgainAndErrors(t *testing.T) {
	ctx := context.Background()
	payload := types.BuildRegistryIndexTaskPayload{Key: "k", RegistryID: 5}

	env := w2rNewEnv(t)
	env.tasks.runAgain = true
	env.tasks.task = w2rTask(t, types.TaskKindBuildRegistryIndex, payload)
	require.NoError(t, env.svc.handleEventExecuteAsyncTask(ctx, w2rEvent()))
	assert.Equal(t, 1, env.producer.count(), "task is announced again")

	env = w2rNewEnv(t)
	env.tasks.runAgain = true
	env.producer.err = errW2rBoom
	env.tasks.task = w2rTask(t, types.TaskKindBuildRegistryIndex, payload)
	require.ErrorIs(t, env.svc.handleEventExecuteAsyncTask(ctx, w2rEvent()), errW2rBoom)

	env = w2rNewEnv(t)
	env.tasks.findErr = errW2rBoom
	require.ErrorIs(t, env.svc.handleEventExecuteAsyncTask(ctx, w2rEvent()), errW2rBoom)

	env = w2rNewEnv(t)
	env.tasks.task = w2rTask(t, types.TaskKindBuildRegistryIndex, payload)
	env.tasks.lockErr = errW2rBoom
	require.Error(t, env.svc.handleEventExecuteAsyncTask(ctx, w2rEvent()))

	env = w2rNewEnv(t)
	env.tasks.task = w2rTask(t, types.TaskKindBuildRegistryIndex, payload)
	env.sources.statusErr = errW2rBoom
	require.NoError(t, env.svc.handleEventExecuteAsyncTask(ctx, w2rEvent()), "final status errors are only logged")
}

func TestW2rStatusUpdateErrors(t *testing.T) {
	ctx := context.Background()
	task := &types.Task{Key: "k"}

	t.Run("processing", func(t *testing.T) {
		for _, mutate := range []func(*w2rEnv){
			func(e *w2rEnv) { e.tasks.lockErr = errW2rBoom },
			func(e *w2rEnv) { e.tasks.updateErr = errW2rBoom },
			func(e *w2rEnv) { e.tasks.runAgainErr = errW2rBoom },
			func(e *w2rEnv) { e.sources.claimErr = errW2rBoom },
		} {
			env := w2rNewEnv(t)
			mutate(env)
			require.Error(t, env.svc.ProcessingStatusUpdate(ctx, task, "run"))
		}
		env := w2rNewEnv(t)
		env.logs.err = errW2rBoom
		require.NoError(t, env.svc.ProcessingStatusUpdate(ctx, task, "run"), "event log failures are ignored")
	})

	t.Run("final", func(t *testing.T) {
		e := w2rEvent()
		for _, processingErr := range []error{nil, errW2rBoom} {
			for _, mutate := range []func(*w2rEnv){
				func(e *w2rEnv) { e.tasks.lockErr = errW2rBoom },
				func(e *w2rEnv) { e.sources.statusErr = errW2rBoom },
				func(e *w2rEnv) { e.tasks.completeErr = errW2rBoom },
			} {
				env := w2rNewEnv(t)
				mutate(env)
				_, err := env.svc.finalStatusUpdate(ctx, e, task, processingErr)
				require.Error(t, err)
			}
			env := w2rNewEnv(t)
			env.logs.err = errW2rBoom
			_, err := env.svc.finalStatusUpdate(ctx, e, task, processingErr)
			require.Error(t, err, "the last logging error is returned by the transaction")
		}
	})
}
