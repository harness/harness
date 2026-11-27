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

package job

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/harness/gitness/lock"
	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

var (
	jobSpanRecorder = tracetest.NewSpanRecorder()
	setJobProvider  sync.Once
	jobProvider     = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(jobSpanRecorder))
)

type handlerFunc func(ctx context.Context, input string, fn ProgressReporter) (string, error)

func (h handlerFunc) Handle(ctx context.Context, input string, fn ProgressReporter) (string, error) {
	return h(ctx, input, fn)
}

type memJobStore struct {
	mu        sync.Mutex
	jobs      map[string]*Job
	findErr   error
	updateErr error
}

func newMemJobStore() *memJobStore {
	return &memJobStore{jobs: make(map[string]*Job)}
}

func (m *memJobStore) Find(_ context.Context, uid string) (*Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.findErr != nil {
		return nil, m.findErr
	}
	j, ok := m.jobs[uid]
	if !ok {
		return nil, store.ErrResourceNotFound
	}
	cp := *j
	return &cp, nil
}

func (m *memJobStore) ListByGroupID(context.Context, string) ([]*Job, error)  { return nil, nil }
func (m *memJobStore) DeleteByGroupID(context.Context, string) (int64, error) { return 0, nil }

func (m *memJobStore) Create(_ context.Context, j *Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *j
	m.jobs[j.UID] = &cp
	return nil
}

func (m *memJobStore) Upsert(ctx context.Context, j *Job) error           { return m.Create(ctx, j) }
func (m *memJobStore) UpdateDefinition(ctx context.Context, j *Job) error { return m.Create(ctx, j) }

func (m *memJobStore) UpdateExecution(_ context.Context, j *Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateErr != nil {
		return m.updateErr
	}
	cp := *j
	m.jobs[j.UID] = &cp
	return nil
}

func (m *memJobStore) UpdateProgress(ctx context.Context, j *Job) error          { return m.Create(ctx, j) }
func (m *memJobStore) CountRunning(context.Context) (int, error)                 { return 0, nil }
func (m *memJobStore) ListReady(context.Context, time.Time, int) ([]*Job, error) { return nil, nil }
func (m *memJobStore) ListDeadlineExceeded(context.Context, time.Time) ([]*Job, error) {
	return nil, nil
}
func (m *memJobStore) NextScheduledTime(context.Context, time.Time) (time.Time, error) {
	return time.Time{}, nil
}
func (m *memJobStore) DeleteOld(context.Context, time.Time) (int64, error) { return 0, nil }
func (m *memJobStore) DeleteByUID(context.Context, string) error           { return nil }

func TestSchedulerTracing(t *testing.T) {
	setJobProvider.Do(func() { otel.SetTracerProvider(jobProvider) })

	st := newMemJobStore()
	ps := pubsub.NewInMemory()
	exec := ProvideExecutor(st, ps)

	var seenTraceID trace.TraceID
	syncHandler := handlerFunc(func(ctx context.Context, _ string, _ ProgressReporter) (string, error) {
		seenTraceID = trace.SpanContextFromContext(ctx).TraceID()
		return "ok", nil
	})
	require.NoError(t, exec.Register("sync", syncHandler))
	require.NoError(t, exec.Register("fail", handlerFunc(func(context.Context, string, ProgressReporter) (string, error) {
		return "", errors.New("boom")
	})))
	exec.finishRegistration()

	sched, err := ProvideScheduler(st, exec, lock.NewInMemory(lock.Config{
		Expiry:     time.Second,
		Tries:      1,
		RetryDelay: time.Millisecond,
	}), ps, Config{
		InstanceID:                  "test-instance",
		Tracing:                     true,
		BackgroundJobsMaxRunning:    2,
		BackgroundJobsRetentionTime: time.Minute,
	})
	require.NoError(t, err)
	sched.done = make(chan struct{})
	close(sched.done)

	before := len(jobSpanRecorder.Ended())
	ctx := context.Background()

	// Schedule single job (success + validation error).
	require.NoError(t, sched.RunJob(ctx, Definition{
		UID:     "job-1",
		Type:    "sync",
		Timeout: 5 * time.Second,
	}))
	require.Error(t, sched.RunJob(ctx, Definition{UID: "", Type: "sync", Timeout: time.Second}))

	// Schedule batch jobs (empty, success, validation error).
	require.NoError(t, sched.RunJobs(ctx, "grp-1", nil))
	require.NoError(t, sched.RunJobs(ctx, "grp-1", []Definition{
		{UID: "job-2", Type: "sync", Timeout: 5 * time.Second},
		{UID: "job-3", Type: "fail", Timeout: 5 * time.Second},
	}))
	require.Error(t, sched.RunJobs(ctx, "grp-1", []Definition{{UID: "", Type: "sync", Timeout: time.Second}}))

	// Execute jobs across success, handler failure, Find error, and UpdateExecution error.
	j2, err := st.Find(ctx, "job-2")
	require.NoError(t, err)
	sched.preExec(j2)
	require.NoError(t, st.UpdateExecution(ctx, j2))
	sched.runJob(ctx, j2)
	sched.WaitJobsDone(ctx)
	assert.True(t, seenTraceID.IsValid())

	j3, err := st.Find(ctx, "job-3")
	require.NoError(t, err)
	sched.preExec(j3)
	require.NoError(t, st.UpdateExecution(ctx, j3))
	sched.runJob(ctx, j3)
	sched.WaitJobsDone(ctx)

	st.updateErr = errors.New("update failed")
	sched.runJob(ctx, j2)
	sched.WaitJobsDone(ctx)
	st.updateErr = nil

	st.findErr = errors.New("find failed")
	sched.runJob(ctx, j2)
	sched.WaitJobsDone(ctx)
	st.findErr = nil

	spans := jobSpanRecorder.Ended()[before:]
	byName := make(map[string][]sdktrace.ReadOnlySpan)
	for _, sp := range spans {
		byName[sp.Name()] = append(byName[sp.Name()], sp)
	}

	require.Len(t, byName["schedule job sync"], 2)
	assert.Equal(t, codes.Unset, byName["schedule job sync"][0].Status().Code)
	assert.Equal(t, codes.Error, byName["schedule job sync"][1].Status().Code)

	require.Len(t, byName["schedule jobs"], 2)
	assert.Contains(t, byName["schedule jobs"][0].Attributes(), attribute.String("job.group_id", "grp-1"))
	assert.Equal(t, codes.Error, byName["schedule jobs"][1].Status().Code)

	require.Len(t, byName["job sync"], 3)
	assert.Contains(t, byName["job sync"][0].Attributes(), attribute.String("job.group_id", "grp-1"))
	assert.Equal(t, codes.Unset, byName["job sync"][0].Status().Code)
	assert.Equal(t, codes.Error, byName["job sync"][1].Status().Code)
	assert.Equal(t, codes.Error, byName["job sync"][2].Status().Code)

	require.Len(t, byName["job fail"], 1)
	assert.Equal(t, codes.Error, byName["job fail"][0].Status().Code)
}
