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

package job

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/harness/gitness/lock"
	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const gtxWait = 10 * time.Second

// gtxStore is an in-memory implementation of Store with optional error injection.
type gtxStore struct {
	mu   sync.Mutex
	jobs map[string]*Job
	errs map[string]error
}

func newGtxStore() *gtxStore {
	return &gtxStore{jobs: map[string]*Job{}, errs: map[string]error{}}
}

func (s *gtxStore) failWith(method string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errs[method] = err
}

func (s *gtxStore) put(j Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[j.UID] = &j
}

func (s *gtxStore) get(uid string) (Job, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[uid]
	if !ok {
		return Job{}, false
	}

	return *j, true
}

func (s *gtxStore) Find(_ context.Context, uid string) (*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.errs["Find"]; err != nil {
		return nil, err
	}
	j, ok := s.jobs[uid]
	if !ok {
		return nil, store.ErrResourceNotFound
	}
	cpy := *j

	return &cpy, nil
}

func (s *gtxStore) ListByGroupID(_ context.Context, groupID string) ([]*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.errs["ListByGroupID"]; err != nil {
		return nil, err
	}
	var out []*Job
	for _, j := range s.jobs {
		if j.GroupID == groupID {
			cpy := *j
			out = append(out, &cpy)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].UID < out[b].UID })

	return out, nil
}

func (s *gtxStore) DeleteByGroupID(_ context.Context, groupID string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.errs["DeleteByGroupID"]; err != nil {
		return 0, err
	}
	var n int64
	for uid, j := range s.jobs {
		if j.GroupID == groupID {
			delete(s.jobs, uid)
			n++
		}
	}

	return n, nil
}

func (s *gtxStore) Create(_ context.Context, job *Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.errs["Create"]; err != nil {
		return err
	}
	if _, ok := s.jobs[job.UID]; ok {
		return store.ErrDuplicate
	}
	cpy := *job
	s.jobs[job.UID] = &cpy

	return nil
}

func (s *gtxStore) Upsert(_ context.Context, job *Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.errs["Upsert"]; err != nil {
		return err
	}
	cpy := *job
	s.jobs[job.UID] = &cpy

	return nil
}

func (s *gtxStore) UpdateDefinition(ctx context.Context, job *Job) error {
	return s.UpdateExecution(ctx, job)
}

func (s *gtxStore) UpdateExecution(_ context.Context, job *Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.errs["UpdateExecution"]; err != nil {
		return err
	}
	if _, ok := s.jobs[job.UID]; !ok {
		return store.ErrResourceNotFound
	}
	cpy := *job
	s.jobs[job.UID] = &cpy

	return nil
}

func (s *gtxStore) UpdateProgress(_ context.Context, job *Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.errs["UpdateProgress"]; err != nil {
		return err
	}
	existing, ok := s.jobs[job.UID]
	if !ok {
		return store.ErrResourceNotFound
	}
	existing.RunProgress = job.RunProgress
	existing.Result = job.Result
	existing.Updated = job.Updated

	return nil
}

func (s *gtxStore) CountRunning(context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.errs["CountRunning"]; err != nil {
		return 0, err
	}
	var n int
	for _, j := range s.jobs {
		if j.State == JobStateRunning {
			n++
		}
	}

	return n, nil
}

func (s *gtxStore) ListReady(_ context.Context, now time.Time, limit int) ([]*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.errs["ListReady"]; err != nil {
		return nil, err
	}
	var out []*Job
	for _, j := range s.jobs {
		if j.State == JobStateScheduled && j.Scheduled <= now.UnixMilli() {
			cpy := *j
			out = append(out, &cpy)
		}
	}
	sort.Slice(out, func(a, b int) bool {
		if out[a].Priority != out[b].Priority {
			return out[a].Priority > out[b].Priority
		}
		if out[a].Scheduled != out[b].Scheduled {
			return out[a].Scheduled < out[b].Scheduled
		}

		return out[a].UID < out[b].UID
	})
	if len(out) > limit {
		out = out[:limit]
	}

	return out, nil
}

func (s *gtxStore) ListDeadlineExceeded(_ context.Context, now time.Time) ([]*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.errs["ListDeadlineExceeded"]; err != nil {
		return nil, err
	}
	var out []*Job
	for _, j := range s.jobs {
		if j.State == JobStateRunning && j.RunDeadline < now.UnixMilli() {
			cpy := *j
			out = append(out, &cpy)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].UID < out[b].UID })

	return out, nil
}

func (s *gtxStore) NextScheduledTime(_ context.Context, now time.Time) (time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.errs["NextScheduledTime"]; err != nil {
		return time.Time{}, err
	}
	var next int64
	for _, j := range s.jobs {
		if j.State == JobStateScheduled && j.Scheduled > now.UnixMilli() && (next == 0 || j.Scheduled < next) {
			next = j.Scheduled
		}
	}
	if next == 0 {
		return time.Time{}, nil
	}

	return time.UnixMilli(next), nil
}

func (s *gtxStore) DeleteOld(_ context.Context, olderThan time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.errs["DeleteOld"]; err != nil {
		return 0, err
	}
	var n int64
	for uid, j := range s.jobs {
		if !j.IsRecurring && j.State.IsCompleted() && j.Updated < olderThan.UnixMilli() {
			delete(s.jobs, uid)
			n++
		}
	}

	return n, nil
}

func (s *gtxStore) DeleteByUID(_ context.Context, uid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.errs["DeleteByUID"]; err != nil {
		return err
	}
	delete(s.jobs, uid)

	return nil
}

// gtxHandler adapts a function to the Handler interface.
type gtxHandler func(ctx context.Context, input string, fn ProgressReporter) (string, error)

func (h gtxHandler) Handle(ctx context.Context, input string, fn ProgressReporter) (string, error) {
	return h(ctx, input, fn)
}

// gtxPublisher records the published messages.
type gtxPublisher struct {
	mu     sync.Mutex
	topics []string
	err    error
}

func (p *gtxPublisher) Publish(_ context.Context, topic string, _ []byte, _ ...pubsub.PublishOption) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.topics = append(p.topics, topic)

	return p.err
}

func (p *gtxPublisher) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return len(p.topics)
}

// gtxFailingManager is a mutex manager that can't create or lock mutexes.
type gtxFailingManager struct{ newErr error }

func (m gtxFailingManager) NewMutex(string, ...lock.Option) (lock.Mutex, error) {
	if m.newErr != nil {
		return nil, m.newErr
	}

	return gtxFailingMutex{}, nil
}

type gtxFailingMutex struct{}

func (gtxFailingMutex) Key() string { return "gtx" }

func (gtxFailingMutex) Lock(context.Context) error {
	return errors.New("gtx lock failure")
}

func (gtxFailingMutex) Unlock(context.Context) error { return nil }

func gtxNewLockManager() lock.MutexManager {
	return lock.NewInMemory(lock.Config{
		App:        "gtx",
		Namespace:  "jobs",
		Expiry:     10 * time.Second,
		Tries:      50,
		RetryDelay: 5 * time.Millisecond,
	})
}

func gtxNewScheduler(t *testing.T, st Store, maxRunning int) (*Scheduler, *Executor) {
	t.Helper()

	executor := NewExecutor(st, pubsub.NewInMemory())
	s, err := NewScheduler(st, executor, gtxNewLockManager(), pubsub.NewInMemory(), "gtx-instance", maxRunning, time.Hour)
	require.NoError(t, err)

	// a closed done channel makes the scheduleProcessing goroutines of a scheduler that doesn't run exit right away.
	s.done = make(chan struct{})
	close(s.done)

	return s, executor
}

func TestGtxDefinition(t *testing.T) {
	valid := Definition{UID: "uid", Type: "type", MaxRetries: 2, Timeout: 5 * time.Second, Data: "payload"}

	tests := []struct {
		name    string
		mutate  func(d *Definition)
		wantErr string
	}{
		{name: "valid", mutate: func(*Definition) {}},
		{name: "no retries", mutate: func(d *Definition) { d.MaxRetries = 0 }},
		{name: "missing type", mutate: func(d *Definition) { d.Type = "" }, wantErr: "Type must not be empty"},
		{name: "missing uid", mutate: func(d *Definition) { d.UID = "" }, wantErr: "unique identifier"},
		{name: "negative retries", mutate: func(d *Definition) { d.MaxRetries = -1 }, wantErr: "MaxRetries"},
		{
			name: "timeout too short", mutate: func(d *Definition) { d.Timeout = time.Second - 1 },
			wantErr: "Timeout too short",
		},
		{name: "minimal timeout", mutate: func(d *Definition) { d.Timeout = time.Second }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			d := valid
			test.mutate(&d)

			err := d.Validate()
			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}

	t.Run("new job", func(t *testing.T) {
		before := time.Now().UnixMilli()
		j := valid.toNewJob()
		after := time.Now().UnixMilli()

		assert.Equal(t, "uid", j.UID)
		assert.Equal(t, "type", j.Type)
		assert.Equal(t, "payload", j.Data)
		assert.Equal(t, 5, j.MaxDurationSeconds)
		assert.Equal(t, 2, j.MaxRetries)
		assert.Equal(t, JobStateScheduled, j.State)
		assert.Equal(t, JobPriorityNormal, j.Priority)
		assert.False(t, j.IsRecurring)
		assert.GreaterOrEqual(t, j.Scheduled, before)
		assert.LessOrEqual(t, j.Scheduled, after)
		assert.Equal(t, j.Scheduled, j.RunDeadline)
		assert.Zero(t, j.TotalExecutions)
		assert.Zero(t, j.LastExecuted)
	})
}

func TestGtxState(t *testing.T) {
	for _, s := range []State{JobStateFinished, JobStateFailed, JobStateCanceled} {
		assert.True(t, s.IsCompleted(), s)
	}
	for _, s := range []State{JobStateScheduled, JobStateRunning, ""} {
		assert.False(t, s.IsCompleted(), s)
	}

	t.Run("enum is sorted and complete", func(t *testing.T) {
		var s State
		assert.Equal(t,
			[]any{JobStateCanceled, JobStateFailed, JobStateFinished, JobStateRunning, JobStateScheduled}, s.Enum())
	})

	t.Run("sanitize", func(t *testing.T) {
		got, ok := JobStateRunning.Sanitize()
		assert.True(t, ok)
		assert.Equal(t, JobStateRunning, got)

		_, ok = State("gtx-unknown").Sanitize()
		assert.False(t, ok)

		all, def := GetAllJobStates()
		assert.Len(t, all, 5)
		assert.Equal(t, State(""), def)
	})

	t.Run("generic sanitize", func(t *testing.T) {
		all := func() ([]string, string) { return []string{"a", "b"}, "def" }

		got, ok := Sanitize("a", all)
		assert.True(t, ok)
		assert.Equal(t, "a", got)

		got, ok = Sanitize("zzz", all)
		assert.False(t, ok)
		assert.NotEqual(t, "zzz", got)
	})
}

func TestGtxExecutor_Register(t *testing.T) {
	h := gtxHandler(func(context.Context, string, ProgressReporter) (string, error) { return "", nil })

	e := NewExecutor(newGtxStore(), &gtxPublisher{})
	require.ErrorContains(t, e.Register("", h), "jobType must not be empty")
	require.ErrorContains(t, e.Register("t", nil), "Handler is nil")
	require.NoError(t, e.Register("t", h))
	require.ErrorContains(t, e.Register("t", h), "already defined")

	e.finishRegistration()
	require.ErrorContains(t, e.Register("other", h), "registration is complete")
}

func TestGtxExecutor_Exec(t *testing.T) {
	ctx := context.Background()

	t.Run("unknown job type", func(t *testing.T) {
		e := NewExecutor(newGtxStore(), &gtxPublisher{})

		_, err := e.exec(ctx, "uid", "unknown", "")
		require.ErrorIs(t, err, errNoHandlerDefined)
	})

	t.Run("result and error of the handler", func(t *testing.T) {
		e := NewExecutor(newGtxStore(), &gtxPublisher{})
		require.NoError(t, e.Register("ok", gtxHandler(
			func(_ context.Context, input string, _ ProgressReporter) (string, error) { return "got " + input, nil })))
		require.NoError(t, e.Register("fail", gtxHandler(
			func(context.Context, string, ProgressReporter) (string, error) {
				return "partial", errors.New("gtx handler failure")
			})))

		got, err := e.exec(ctx, "uid", "ok", "in")
		require.NoError(t, err)
		assert.Equal(t, "got in", got)

		got, err = e.exec(ctx, "uid", "fail", "")
		require.ErrorContains(t, err, "gtx handler failure")
		assert.Equal(t, "partial", got)
	})

	t.Run("panic is converted to an error", func(t *testing.T) {
		e := NewExecutor(newGtxStore(), &gtxPublisher{})
		require.NoError(t, e.Register("panic", gtxHandler(
			func(context.Context, string, ProgressReporter) (string, error) { panic("gtx boom") })))

		_, err := e.exec(ctx, "uid-1", "panic", "")
		require.ErrorContains(t, err, "panic while processing job=uid-1 type=panic: gtx boom")
	})

	t.Run("progress reporter", func(t *testing.T) {
		st := newGtxStore()
		st.put(Job{UID: "uid", Type: "progress", State: JobStateRunning})
		pub := &gtxPublisher{}

		e := NewExecutor(st, pub)
		var reportErrs []error
		require.NoError(t, e.Register("progress", gtxHandler(
			func(_ context.Context, _ string, fn ProgressReporter) (string, error) {
				reportErrs = append(reportErrs, fn(50, "halfway"), fn(-1, ""), fn(101, ""), fn(ProgressMin, "start"))
				return "done", nil
			})))

		_, err := e.exec(ctx, "uid", "progress", "")
		require.NoError(t, err)

		require.Len(t, reportErrs, 4)
		require.NoError(t, reportErrs[0])
		require.ErrorContains(t, reportErrs[1], "progress must be between 0 and 100")
		require.ErrorContains(t, reportErrs[2], "progress must be between 0 and 100")
		require.NoError(t, reportErrs[3])

		j, ok := st.get("uid")
		require.True(t, ok)
		assert.Equal(t, ProgressMin, j.RunProgress)
		assert.Equal(t, "start", j.Result)
		assert.Equal(t, 2, pub.count(), "state change is published for every valid report")
	})

	t.Run("progress reporter surfaces store and publish failures", func(t *testing.T) {
		st := newGtxStore()
		st.put(Job{UID: "uid", Type: "progress", State: JobStateRunning})
		pub := &gtxPublisher{err: errors.New("gtx publish failure")}

		e := NewExecutor(st, pub)
		var reportErrs []error
		require.NoError(t, e.Register("progress", gtxHandler(
			func(_ context.Context, _ string, fn ProgressReporter) (string, error) {
				// a publish failure is only logged.
				reportErrs = append(reportErrs, fn(10, "x"))
				st.failWith("UpdateProgress", errors.New("gtx store failure"))
				reportErrs = append(reportErrs, fn(20, "y"))
				return "", nil
			})))

		_, err := e.exec(ctx, "uid", "progress", "")
		require.NoError(t, err)
		require.NoError(t, reportErrs[0])
		require.ErrorContains(t, reportErrs[1], "gtx store failure")
	})

	t.Run("fail progress", func(t *testing.T) {
		assert.Equal(t, Progress{State: JobStateFailed, Progress: ProgressMax}, FailProgress())
	})
}

func TestGtxStateChangeEncoding(t *testing.T) {
	ctx := context.Background()

	j := &Job{
		UID: "uid", Type: "type", State: JobStateFailed, RunProgress: 42, Result: "res", LastFailureError: "err",
	}

	payload, err := encodeStateChange(j)
	require.NoError(t, err)

	got, err := DecodeStateChange(payload)
	require.NoError(t, err)
	assert.Equal(t, &StateChange{
		UID: "uid", Type: "type", State: JobStateFailed, Progress: 42, Result: "res", Failure: "err",
	}, got)

	_, err = DecodeStateChange([]byte("garbage"))
	require.Error(t, err)

	t.Run("publish", func(t *testing.T) {
		pub := &gtxPublisher{}
		require.NoError(t, publishStateChange(ctx, pub, j))
		assert.Equal(t, []string{PubSubTopicStateChange}, pub.topics)
	})

	t.Run("publish failure", func(t *testing.T) {
		pub := &gtxPublisher{err: errors.New("gtx publish failure")}
		require.ErrorContains(t, publishStateChange(ctx, pub, j), "gtx publish failure")
	})
}

func TestGtxPostExec(t *testing.T) {
	tests := []struct {
		name         string
		job          Job
		result       string
		failure      string
		wantState    State
		wantFailures int
		wantErrText  string
		wantResched  bool
	}{
		{
			name: "success", job: Job{State: JobStateRunning, RunBy: "me"}, result: "res",
			wantState: JobStateFinished,
		},
		{
			name: "failure without retries", job: Job{State: JobStateRunning}, failure: "boom",
			wantState: JobStateFailed, wantFailures: 1, wantErrText: "boom",
		},
		{
			name: "failure with retries is rescheduled", job: Job{State: JobStateRunning, MaxRetries: 1}, failure: "boom",
			wantState: JobStateScheduled, wantFailures: 1, wantErrText: "boom", wantResched: true,
		},
		{
			name:    "failure after the retries are used up",
			job:     Job{State: JobStateRunning, MaxRetries: 1, ConsecutiveFailures: 1},
			failure: "boom", wantState: JobStateFailed, wantFailures: 2, wantErrText: "boom",
		},
		{
			name: "canceled job that succeeded is finished", job: Job{State: JobStateCanceled}, result: "res",
			wantState: JobStateFinished,
		},
		{
			name: "canceled job that failed stays canceled", job: Job{State: JobStateCanceled}, failure: "boom",
			wantState: JobStateCanceled,
		},
		{name: "scheduled job is left alone", job: Job{State: JobStateScheduled}, wantState: JobStateScheduled},
		{name: "finished job is left alone", job: Job{State: JobStateFinished}, failure: "x", wantState: JobStateFinished},
		{
			name:      "recurring success is rescheduled",
			job:       Job{State: JobStateRunning, IsRecurring: true, RecurringCron: "*/5 * * * *", ConsecutiveFailures: 3},
			wantState: JobStateScheduled, wantResched: true,
		},
		{
			name:      "recurring failure keeps the failure count",
			job:       Job{State: JobStateRunning, IsRecurring: true, RecurringCron: "*/5 * * * *"},
			failure:   "boom",
			wantState: JobStateScheduled, wantFailures: 1, wantErrText: "boom", wantResched: true,
		},
		{
			name:      "recurring with invalid cron fails",
			job:       Job{State: JobStateRunning, IsRecurring: true, RecurringCron: "not a cron"},
			wantState: JobStateFailed, wantErrText: "failed to parse cron string",
		},
		{
			name:      "recurring with invalid cron keeps the earlier failure",
			job:       Job{State: JobStateRunning, IsRecurring: true, RecurringCron: "not a cron"},
			failure:   "boom",
			wantState: JobStateFailed, wantFailures: 1, wantErrText: "; boom",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			j := test.job
			now := time.Now().UnixMilli()

			postExec(&j, test.result, test.failure)

			assert.Equal(t, test.wantState, j.State)
			assert.Equal(t, test.wantFailures, j.ConsecutiveFailures)
			if test.wantErrText != "" {
				assert.Contains(t, j.LastFailureError, test.wantErrText)
			}
			if test.wantResched {
				assert.Greater(t, j.Scheduled, now)
			}
			if test.job.State == JobStateRunning && test.wantState == JobStateFinished {
				assert.Equal(t, ProgressMax, j.RunProgress)
				assert.Equal(t, test.result, j.Result)
				assert.Empty(t, j.RunBy)
			}
		})
	}
}

func TestGtxScheduler_PreExec(t *testing.T) {
	s, _ := gtxNewScheduler(t, newGtxStore(), 1)

	j := &Job{
		MaxDurationSeconds: 0, TotalExecutions: 2, Result: "old", LastFailureError: "old", RunProgress: 70,
	}
	before := time.Now().UnixMilli()
	s.preExec(j)

	assert.Equal(t, JobStateRunning, j.State)
	assert.Equal(t, 1, j.MaxDurationSeconds, "duration is at least a second")
	assert.Equal(t, 3, j.TotalExecutions)
	assert.Equal(t, "gtx-instance", j.RunBy)
	assert.Empty(t, j.Result)
	assert.Empty(t, j.LastFailureError)
	assert.Equal(t, ProgressMin, j.RunProgress)
	assert.GreaterOrEqual(t, j.LastExecuted, before)
	assert.GreaterOrEqual(t, j.RunDeadline, before+1000)
}

func TestGtxNewScheduler(t *testing.T) {
	s, err := NewScheduler(newGtxStore(), nil, nil, nil, "id", -5, time.Hour)
	require.NoError(t, err)
	assert.Equal(t, 1, s.maxRunning)

	s, err = NewScheduler(newGtxStore(), nil, nil, nil, "id", 7, time.Hour)
	require.NoError(t, err)
	assert.Equal(t, 7, s.maxRunning)
	assert.Equal(t, "id", s.instanceID)

	exec := ProvideExecutor(newGtxStore(), pubsub.NewInMemory())
	require.NotNil(t, exec)

	provided, err := ProvideScheduler(newGtxStore(), exec, gtxNewLockManager(), pubsub.NewInMemory(), Config{
		InstanceID: "cfg-instance", BackgroundJobsMaxRunning: 3, BackgroundJobsRetentionTime: time.Minute,
	})
	require.NoError(t, err)
	assert.Equal(t, "cfg-instance", provided.instanceID)
	assert.Equal(t, 3, provided.maxRunning)
	assert.Equal(t, time.Minute, provided.retentionTime)
}

func TestGtxScheduler_RunJobs(t *testing.T) {
	ctx := context.Background()
	st := newGtxStore()
	s, _ := gtxNewScheduler(t, st, 1)

	t.Run("run job", func(t *testing.T) {
		require.NoError(t, s.RunJob(ctx, Definition{UID: "j1", Type: "t", Timeout: time.Second, Data: "d"}))

		j, ok := st.get("j1")
		require.True(t, ok)
		assert.Equal(t, JobStateScheduled, j.State)
		assert.Equal(t, "d", j.Data)
	})

	t.Run("invalid definition", func(t *testing.T) {
		require.Error(t, s.RunJob(ctx, Definition{UID: "j2", Timeout: time.Second}))
		_, ok := st.get("j2")
		assert.False(t, ok)
	})

	t.Run("duplicate", func(t *testing.T) {
		err := s.RunJob(ctx, Definition{UID: "j1", Type: "t", Timeout: time.Second})
		require.ErrorIs(t, err, store.ErrDuplicate)
		require.ErrorContains(t, err, "failed to add new job to the database")
	})

	t.Run("run jobs in a group", func(t *testing.T) {
		require.NoError(t, s.RunJobs(ctx, "group", nil))

		require.NoError(t, s.RunJobs(ctx, "group", []Definition{
			{UID: "g1", Type: "t", Timeout: time.Second},
			{UID: "g2", Type: "t", Timeout: time.Second},
		}))

		progress, err := s.GetJobProgressForGroup(ctx, "group")
		require.NoError(t, err)
		assert.Equal(t, []Progress{{State: JobStateScheduled}, {State: JobStateScheduled}}, progress)

		j, ok := st.get("g1")
		require.True(t, ok)
		assert.Equal(t, "group", j.GroupID)
	})

	t.Run("run jobs with an invalid definition", func(t *testing.T) {
		err := s.RunJobs(ctx, "invalid-group", []Definition{
			{UID: "i1", Type: "t", Timeout: time.Second},
			{UID: "i2", Timeout: time.Second},
		})
		require.Error(t, err)
		_, ok := st.get("i1")
		assert.False(t, ok, "nothing is created if any definition is invalid")
	})

	t.Run("run jobs with a store failure", func(t *testing.T) {
		failing := newGtxStore()
		failing.failWith("Create", errors.New("gtx store failure"))
		fs, _ := gtxNewScheduler(t, failing, 1)

		err := fs.RunJobs(ctx, "g", []Definition{{UID: "x", Type: "t", Timeout: time.Second}})
		require.ErrorContains(t, err, "gtx store failure")
		require.ErrorContains(t, fs.RunJob(ctx, Definition{UID: "x", Type: "t", Timeout: time.Second}), "gtx store failure")
	})

	t.Run("progress", func(t *testing.T) {
		st.put(Job{UID: "p1", State: JobStateRunning, RunProgress: 30, Result: "r", LastFailureError: "f"})

		got, err := s.GetJobProgress(ctx, "p1")
		require.NoError(t, err)
		assert.Equal(t, Progress{State: JobStateRunning, Progress: 30, Result: "r", Failure: "f"}, got)

		_, err = s.GetJobProgress(ctx, "missing")
		require.ErrorIs(t, err, store.ErrResourceNotFound)

		st.failWith("ListByGroupID", errors.New("gtx store failure"))
		_, err = s.GetJobProgressForGroup(ctx, "group")
		require.ErrorContains(t, err, "gtx store failure")
	})

	t.Run("purge", func(t *testing.T) {
		n, err := s.PurgeJobsByGroupID(ctx, "group")
		require.NoError(t, err)
		assert.Equal(t, int64(2), n)

		require.NoError(t, s.PurgeJobByUID(ctx, "j1"))
		_, ok := st.get("j1")
		assert.False(t, ok)

		st.failWith("DeleteByGroupID", errors.New("gtx store failure"))
		_, err = s.PurgeJobsByGroupID(ctx, "group")
		require.ErrorContains(t, err, "failed to delete jobs by group id=group")

		st.failWith("DeleteByUID", errors.New("gtx store failure"))
		require.ErrorContains(t, s.PurgeJobByUID(ctx, "j1"), "failed to delete job with id=j1")
	})
}

func TestGtxMapToProgress(t *testing.T) {
	assert.Nil(t, mapToProgressMany(nil))
	assert.Empty(t, mapToProgressMany([]*Job{}))
	assert.Equal(t,
		[]Progress{{State: JobStateFinished, Progress: 100, Result: "ok"}},
		mapToProgressMany([]*Job{{State: JobStateFinished, RunProgress: 100, Result: "ok"}}))
}

func TestGtxScheduler_AddRecurring(t *testing.T) {
	ctx := context.Background()
	st := newGtxStore()
	s, _ := gtxNewScheduler(t, st, 1)

	require.NoError(t, s.AddRecurring(ctx, "rec", "type", "*/10 * * * *", 30*time.Second))

	j, ok := st.get("rec")
	require.True(t, ok)
	assert.True(t, j.IsRecurring)
	assert.Equal(t, "*/10 * * * *", j.RecurringCron)
	assert.Equal(t, 30, j.MaxDurationSeconds)
	assert.Equal(t, JobPriorityElevated, j.Priority)
	assert.Equal(t, JobStateScheduled, j.State)
	assert.Greater(t, j.Scheduled, time.Now().UnixMilli()-1)
	assert.LessOrEqual(t, j.Scheduled, time.Now().Add(10*time.Minute).UnixMilli())

	err := s.AddRecurring(ctx, "bad", "type", "not a cron", time.Second)
	require.ErrorContains(t, err, "invalid cron definition string for job type=type")

	st.failWith("Upsert", errors.New("gtx store failure"))
	err = s.AddRecurring(ctx, "rec2", "type", "* * * * *", time.Second)
	require.ErrorContains(t, err, "failed to upsert job id=rec2 type=type")
}

func TestGtxGlobalLock(t *testing.T) {
	ctx := context.Background()

	t.Run("lock and unlock", func(t *testing.T) {
		mx, err := globalLock(ctx, gtxNewLockManager())
		require.NoError(t, err)
		assert.Equal(t, "gtx:jobs:jobs", mx.Key())
		require.NoError(t, mx.Unlock(ctx))
	})

	t.Run("mutex creation fails", func(t *testing.T) {
		_, err := globalLock(ctx, gtxFailingManager{newErr: errors.New("gtx failure")})
		require.ErrorContains(t, err, "gtx failure")
	})

	t.Run("lock fails", func(t *testing.T) {
		_, err := globalLock(ctx, gtxFailingManager{})
		require.ErrorContains(t, err, "gtx lock failure")
	})
}

func TestGtxJobPurge(t *testing.T) {
	ctx := context.Background()
	old := time.Now().Add(-48 * time.Hour).UnixMilli()
	recent := time.Now().UnixMilli()

	newStore := func() *gtxStore {
		st := newGtxStore()
		st.put(Job{UID: "old-finished", State: JobStateFinished, Updated: old})
		st.put(Job{UID: "old-failed", State: JobStateFailed, Updated: old})
		st.put(Job{UID: "old-running", State: JobStateRunning, Updated: old})
		st.put(Job{UID: "old-recurring", State: JobStateFinished, IsRecurring: true, Updated: old})
		st.put(Job{UID: "recent-finished", State: JobStateFinished, Updated: recent})

		return st
	}

	t.Run("deletes old completed jobs", func(t *testing.T) {
		st := newStore()
		got, err := newJobPurge(st, gtxNewLockManager(), 24*time.Hour).Handle(ctx, "", nil)
		require.NoError(t, err)
		assert.Equal(t, "deleted 2 old jobs", got)

		for _, uid := range []string{"old-finished", "old-failed"} {
			_, ok := st.get(uid)
			assert.False(t, ok, uid)
		}
		for _, uid := range []string{"old-running", "old-recurring", "recent-finished"} {
			_, ok := st.get(uid)
			assert.True(t, ok, uid)
		}
	})

	t.Run("nothing to delete", func(t *testing.T) {
		got, err := newJobPurge(newStore(), gtxNewLockManager(), 1000*time.Hour).Handle(ctx, "", nil)
		require.NoError(t, err)
		assert.Equal(t, "no old jobs found", got)
	})

	t.Run("negative age is treated as zero", func(t *testing.T) {
		assert.Zero(t, newJobPurge(newStore(), gtxNewLockManager(), -time.Hour).minOldAge)
	})

	t.Run("lock failure", func(t *testing.T) {
		_, err := newJobPurge(newStore(), gtxFailingManager{}, time.Hour).Handle(ctx, "", nil)
		require.ErrorContains(t, err, "failed to obtain the lock to clean up old jobs")
	})

	t.Run("store failure", func(t *testing.T) {
		st := newStore()
		st.failWith("DeleteOld", errors.New("gtx store failure"))

		_, err := newJobPurge(st, gtxNewLockManager(), time.Hour).Handle(ctx, "", nil)
		require.ErrorContains(t, err, "failed to purge old jobs")
	})
}

func TestGtxJobOverdue(t *testing.T) {
	ctx := context.Background()
	pastDeadline := time.Now().Add(-time.Hour).UnixMilli()
	futureDeadline := time.Now().Add(time.Hour).UnixMilli()

	newScheduler := func(st Store) *Scheduler {
		s, _ := gtxNewScheduler(t, st, 1)
		s.done = make(chan struct{})
		s.signal = make(chan time.Time, 1)

		return s
	}

	t.Run("no overdue jobs", func(t *testing.T) {
		st := newGtxStore()
		st.put(Job{UID: "running", State: JobStateRunning, RunDeadline: futureDeadline})

		got, err := newJobOverdue(st, gtxNewLockManager(), newScheduler(st)).Handle(ctx, "", nil)
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("overdue jobs fail or are rescheduled", func(t *testing.T) {
		st := newGtxStore()
		st.put(Job{UID: "no-retries", State: JobStateRunning, RunDeadline: pastDeadline})
		st.put(Job{UID: "with-retries", State: JobStateRunning, RunDeadline: pastDeadline, MaxRetries: 2})
		st.put(Job{UID: "in-time", State: JobStateRunning, RunDeadline: futureDeadline})

		s := newScheduler(st)
		got, err := newJobOverdue(st, gtxNewLockManager(), s).Handle(ctx, "", nil)
		require.NoError(t, err)
		assert.Equal(t, "found 2 overdue jobs", got)

		failed, _ := st.get("no-retries")
		assert.Equal(t, JobStateFailed, failed.State)
		assert.Equal(t, "deadline exceeded", failed.LastFailureError)

		retried, _ := st.get("with-retries")
		assert.Equal(t, JobStateScheduled, retried.State)
		assert.Equal(t, 1, retried.ConsecutiveFailures)

		untouched, _ := st.get("in-time")
		assert.Equal(t, JobStateRunning, untouched.State)

		// the scheduler is told when to look at the rescheduled job.
		select {
		case scheduled := <-s.signal:
			assert.Equal(t, retried.Scheduled, scheduled.UnixMilli())
		case <-time.After(gtxWait):
			t.Fatal("scheduler wasn't signaled")
		}
	})

	t.Run("lock failure", func(t *testing.T) {
		st := newGtxStore()
		_, err := newJobOverdue(st, gtxFailingManager{}, newScheduler(st)).Handle(ctx, "", nil)
		require.ErrorContains(t, err, "failed to obtain the lock to reclaim overdue jobs")
	})

	t.Run("list failure", func(t *testing.T) {
		st := newGtxStore()
		st.failWith("ListDeadlineExceeded", errors.New("gtx store failure"))

		_, err := newJobOverdue(st, gtxNewLockManager(), newScheduler(st)).Handle(ctx, "", nil)
		require.ErrorContains(t, err, "failed to list overdue jobs")
	})

	t.Run("update failure", func(t *testing.T) {
		st := newGtxStore()
		st.put(Job{UID: "overdue", State: JobStateRunning, RunDeadline: pastDeadline})
		st.failWith("UpdateExecution", errors.New("gtx store failure"))

		_, err := newJobOverdue(st, gtxNewLockManager(), newScheduler(st)).Handle(ctx, "", nil)
		require.ErrorContains(t, err, "failed update overdue job")
	})
}

func TestGtxScheduler_CancelJob(t *testing.T) {
	ctx := context.Background()

	t.Run("lock failure", func(t *testing.T) {
		st := newGtxStore()
		s, _ := gtxNewScheduler(t, st, 1)
		s.mxManager = gtxFailingManager{}

		require.ErrorContains(t, s.CancelJob(ctx, "x"), "failed to obtain global lock to cancel a job")
	})

	t.Run("unknown job", func(t *testing.T) {
		s, _ := gtxNewScheduler(t, newGtxStore(), 1)
		require.NoError(t, s.CancelJob(ctx, "missing"))
	})

	t.Run("store failure", func(t *testing.T) {
		st := newGtxStore()
		st.failWith("Find", errors.New("gtx store failure"))
		s, _ := gtxNewScheduler(t, st, 1)

		require.ErrorContains(t, s.CancelJob(ctx, "x"), "failed to find job to cancel")
	})

	t.Run("recurring jobs can't be canceled", func(t *testing.T) {
		st := newGtxStore()
		st.put(Job{UID: "rec", State: JobStateScheduled, IsRecurring: true})
		s, _ := gtxNewScheduler(t, st, 1)

		require.ErrorContains(t, s.CancelJob(ctx, "rec"), "can't cancel recurring jobs")
	})

	t.Run("completed jobs are ignored", func(t *testing.T) {
		st := newGtxStore()
		s, _ := gtxNewScheduler(t, st, 1)

		for _, state := range []State{JobStateFinished, JobStateFailed, JobStateCanceled} {
			st.put(Job{UID: string(state), State: state})
			require.NoError(t, s.CancelJob(ctx, string(state)))

			j, _ := st.get(string(state))
			assert.Equal(t, state, j.State)
		}
	})

	t.Run("scheduled job is canceled and other instances are notified", func(t *testing.T) {
		st := newGtxStore()
		st.put(Job{UID: "scheduled", State: JobStateScheduled})
		s, _ := gtxNewScheduler(t, st, 1)
		// the default send timeout of the in-memory pubsub is too short to be relied on.
		ps := pubsub.NewInMemory(pubsub.WithSendTimeout(gtxWait))
		s.pubsubService = ps

		received := make(chan string, 1)
		consumer := ps.Subscribe(ctx, PubSubTopicCancelJob, func(payload []byte) error {
			received <- string(payload)
			return nil
		})
		t.Cleanup(func() { _ = consumer.Close() })

		require.NoError(t, s.CancelJob(ctx, "scheduled"))

		j, _ := st.get("scheduled")
		assert.Equal(t, JobStateCanceled, j.State)

		select {
		case got := <-received:
			assert.Equal(t, "scheduled", got)
		case <-time.After(gtxWait):
			t.Fatal("cancel message wasn't published")
		}
	})

	t.Run("running job on this instance is canceled directly", func(t *testing.T) {
		st := newGtxStore()
		st.put(Job{UID: "running", State: JobStateRunning})
		s, _ := gtxNewScheduler(t, st, 1)

		canceled := make(chan struct{})
		s.cancelJobMap["running"] = func() { close(canceled) }

		require.NoError(t, s.CancelJob(ctx, "running"))

		select {
		case <-canceled:
		case <-time.After(gtxWait):
			t.Fatal("job context wasn't canceled")
		}
	})

	t.Run("update failure", func(t *testing.T) {
		st := newGtxStore()
		st.put(Job{UID: "scheduled", State: JobStateScheduled})
		st.failWith("UpdateExecution", errors.New("gtx store failure"))
		s, _ := gtxNewScheduler(t, st, 1)

		require.ErrorContains(t, s.CancelJob(ctx, "scheduled"), "failed to update job to cancel it")
	})

	t.Run("cancel message handler", func(t *testing.T) {
		s, _ := gtxNewScheduler(t, newGtxStore(), 1)

		canceled := make(chan struct{}, 1)
		s.cancelJobMap["job"] = func() { canceled <- struct{}{} }

		require.NoError(t, s.handleCancelJob(nil))
		require.NoError(t, s.handleCancelJob([]byte("other")))
		assert.Empty(t, canceled)

		require.NoError(t, s.handleCancelJob([]byte("job")))
		assert.Len(t, canceled, 1)
	})
}

func TestGtxScheduler_ProcessReadyJobs(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	t.Run("runs the jobs that fit and reports that more are available", func(t *testing.T) {
		st := newGtxStore()
		for _, uid := range []string{"a", "b", "c"} {
			st.put(Job{
				UID: uid, Type: "gtx:wait", State: JobStateScheduled, Scheduled: now.Add(-time.Minute).UnixMilli(),
				MaxDurationSeconds: 10,
			})
		}

		s, executor := gtxNewScheduler(t, st, 2)
		release := make(chan struct{})
		started := make(chan string, 4)
		require.NoError(t, executor.Register("gtx:wait", gtxHandler(
			func(_ context.Context, _ string, _ ProgressReporter) (string, error) {
				started <- "x"
				<-release
				return "ok", nil
			})))

		count, next, gotAll, err := s.processReadyJobs(ctx, now)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
		assert.False(t, gotAll)
		assert.True(t, next.IsZero())

		<-started
		<-started
		running, err := st.CountRunning(ctx)
		require.NoError(t, err)
		assert.Equal(t, 2, running)

		// no free slot: nothing is started
		count, _, gotAll, err = s.processReadyJobs(ctx, now)
		require.NoError(t, err)
		assert.Zero(t, count)
		assert.False(t, gotAll)

		close(release)
		s.WaitJobsDone(ctx)

		finished := 0
		for _, uid := range []string{"a", "b", "c"} {
			if j, _ := st.get(uid); j.State == JobStateFinished {
				finished++
			}
		}
		assert.Equal(t, 2, finished)
	})

	t.Run("all jobs fit and the next execution time is known", func(t *testing.T) {
		st := newGtxStore()
		future := now.Add(time.Hour)
		st.put(Job{UID: "future", Type: "gtx:noop", State: JobStateScheduled, Scheduled: future.UnixMilli()})

		s, executor := gtxNewScheduler(t, st, 5)
		require.NoError(t, executor.Register("gtx:noop", gtxHandler(
			func(context.Context, string, ProgressReporter) (string, error) { return "", nil })))

		count, next, gotAll, err := s.processReadyJobs(ctx, now)
		require.NoError(t, err)
		assert.Zero(t, count)
		assert.True(t, gotAll)
		assert.Equal(t, future.UnixMilli(), next.UnixMilli())
	})

	t.Run("failures", func(t *testing.T) {
		tests := []struct {
			name    string
			method  string
			wantMsg string
		}{
			{name: "count running", method: "CountRunning", wantMsg: "failed to count available slots"},
			{name: "list ready", method: "ListReady", wantMsg: "failed to load scheduled jobs"},
			{name: "next scheduled time", method: "NextScheduledTime", wantMsg: "failed to read next scheduled time"},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				st := newGtxStore()
				st.failWith(test.method, errors.New("gtx store failure"))
				s, _ := gtxNewScheduler(t, st, 5)

				_, _, _, err := s.processReadyJobs(ctx, now)
				require.ErrorContains(t, err, test.wantMsg)
			})
		}

		t.Run("lock", func(t *testing.T) {
			s, _ := gtxNewScheduler(t, newGtxStore(), 5)
			s.mxManager = gtxFailingManager{}

			_, _, _, err := s.processReadyJobs(ctx, now)
			require.ErrorContains(t, err, "failed to obtain global lock")
		})

		t.Run("marking the job as running fails", func(t *testing.T) {
			st := newGtxStore()
			st.put(Job{UID: "a", Type: "gtx:noop", State: JobStateScheduled, Scheduled: now.Add(-time.Minute).UnixMilli()})
			st.failWith("UpdateExecution", errors.New("gtx store failure"))
			s, _ := gtxNewScheduler(t, st, 5)

			count, next, gotAll, err := s.processReadyJobs(ctx, now)
			require.NoError(t, err)
			assert.Zero(t, count)
			assert.False(t, gotAll)
			assert.True(t, next.IsZero())
		})
	})

	t.Run("available slots", func(t *testing.T) {
		st := newGtxStore()
		st.put(Job{UID: "r1", State: JobStateRunning})
		st.put(Job{UID: "r2", State: JobStateRunning})
		s, _ := gtxNewScheduler(t, st, 1)

		slots, err := s.availableSlots(ctx)
		require.NoError(t, err)
		assert.Zero(t, slots, "never negative")
	})
}

func TestGtxScheduler_Run(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	st := newGtxStore()
	s, executor := gtxNewScheduler(t, st, 5)
	s.done = nil // the helper closes it for schedulers that don't run.

	released := make(chan struct{})
	started := make(chan struct{})
	require.NoError(t, executor.Register("gtx:echo", gtxHandler(
		func(_ context.Context, input string, fn ProgressReporter) (string, error) {
			if err := fn(40, "working"); err != nil {
				return "", err
			}
			return "echo:" + input, nil
		})))
	require.NoError(t, executor.Register("gtx:fail", gtxHandler(
		func(context.Context, string, ProgressReporter) (string, error) {
			return "", errors.New("gtx job failure")
		})))
	require.NoError(t, executor.Register("gtx:panic", gtxHandler(
		func(context.Context, string, ProgressReporter) (string, error) { panic("gtx job panic") })))
	require.NoError(t, executor.Register("gtx:block", gtxHandler(
		func(jobCtx context.Context, _ string, _ ProgressReporter) (string, error) {
			close(started)
			select {
			case <-jobCtx.Done():
				return "", jobCtx.Err()
			case <-released:
				return "released", nil
			}
		})))

	// jobs that are ready when the scheduler starts are picked up by its first iteration.
	ready := time.Now().Add(-time.Second).UnixMilli()
	for uid, def := range map[string]Job{
		"echo":  {Type: "gtx:echo", Data: "hello"},
		"fail":  {Type: "gtx:fail"},
		"panic": {Type: "gtx:panic"},
		"block": {Type: "gtx:block"},
		"nohnd": {Type: "gtx:unregistered"},
	} {
		def.UID = uid
		def.State = JobStateScheduled
		def.Scheduled = ready
		def.MaxDurationSeconds = 60
		st.put(def)
	}

	runDone := make(chan error, 1)
	go func() { runDone <- s.Run(ctx) }()

	waitState := func(uid string, state State) Job {
		t.Helper()

		var got Job
		require.Eventually(t, func() bool {
			got, _ = st.get(uid)
			return got.State == state
		}, gtxWait, 10*time.Millisecond, "job %s didn't reach state %s", uid, state)

		return got
	}

	echo := waitState("echo", JobStateFinished)
	assert.Equal(t, "echo:hello", echo.Result)
	assert.Equal(t, ProgressMax, echo.RunProgress)
	assert.Equal(t, 1, echo.TotalExecutions)
	assert.Empty(t, echo.RunBy)

	failed := waitState("fail", JobStateFailed)
	assert.Equal(t, "gtx job failure", failed.LastFailureError)
	assert.Equal(t, 1, failed.ConsecutiveFailures)

	panicked := waitState("panic", JobStateFailed)
	assert.Contains(t, panicked.LastFailureError, "panic while processing job=panic")

	noHandler := waitState("nohnd", JobStateFailed)
	assert.Contains(t, noHandler.LastFailureError, errNoHandlerDefined.Error())

	// the recurring maintenance jobs are created and the scheduler's own handlers are registered.
	for _, uid := range []string{jobUIDPurge, jobUIDOverdue} {
		j, ok := st.get(uid)
		require.True(t, ok, uid)
		assert.True(t, j.IsRecurring)
	}
	require.ErrorContains(t, executor.Register("gtx:late", gtxHandler(nil)), "registration is complete")
	assert.Contains(t, executor.handlerMap, jobTypeOverdue)
	assert.Contains(t, executor.handlerMap, jobTypePurge)

	// starting twice is not possible.
	require.ErrorContains(t, s.Run(ctx), "already started")

	// cancel the running job.
	<-started
	waitState("block", JobStateRunning)
	require.NoError(t, s.CancelJob(ctx, "block"))
	canceled := waitState("block", JobStateCanceled)
	assert.Empty(t, canceled.Result)

	// a job that is added while the scheduler runs is picked up through the signal.
	require.NoError(t, s.RunJob(ctx, Definition{UID: "late", Type: "gtx:echo", Data: "late", Timeout: 30 * time.Second}))
	late := waitState("late", JobStateFinished)
	assert.Equal(t, "echo:late", late.Result)

	cancel()
	select {
	case err := <-runDone:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(gtxWait):
		t.Fatal("scheduler didn't stop")
	}

	waitCtx, waitCancel := context.WithTimeout(context.Background(), gtxWait)
	defer waitCancel()
	s.WaitJobsDone(waitCtx)
	assert.NoError(t, waitCtx.Err())
}

func TestGtxScheduler_RunFailures(t *testing.T) {
	t.Run("necessary jobs can't be created", func(t *testing.T) {
		st := newGtxStore()
		st.failWith("Upsert", errors.New("gtx store failure"))
		s, _ := gtxNewScheduler(t, st, 1)
		s.done = nil

		err := s.Run(context.Background())
		require.ErrorContains(t, err, "failed to create necessary jobs")
	})

	t.Run("lock failure while creating the necessary jobs", func(t *testing.T) {
		s, _ := gtxNewScheduler(t, newGtxStore(), 1)
		s.done = nil
		s.mxManager = gtxFailingManager{}

		err := s.Run(context.Background())
		require.ErrorContains(t, err, "failed to obtain global lock to create necessary jobs")
	})

	t.Run("internal job types are already registered", func(t *testing.T) {
		s, executor := gtxNewScheduler(t, newGtxStore(), 1)
		s.done = nil
		require.NoError(t, executor.Register(jobTypeOverdue, gtxHandler(nil)))

		err := s.Run(context.Background())
		require.ErrorContains(t, err, "failed to register scheduler's internal jobs")
	})
}

func TestGtxScheduler_WaitJobsDone(t *testing.T) {
	t.Run("interrupted by the context", func(t *testing.T) {
		s, _ := gtxNewScheduler(t, newGtxStore(), 1)
		s.wgRunning.Add(1)
		t.Cleanup(s.wgRunning.Done)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		start := time.Now()
		s.WaitJobsDone(ctx)
		assert.Less(t, time.Since(start), gtxWait)
		require.Error(t, ctx.Err())
	})

	t.Run("nothing running", func(t *testing.T) {
		s, _ := gtxNewScheduler(t, newGtxStore(), 1)

		ctx, cancel := context.WithTimeout(context.Background(), gtxWait)
		defer cancel()

		s.WaitJobsDone(ctx)
		require.NoError(t, ctx.Err())
	})
}

func TestGtxScheduleProcessing(t *testing.T) {
	s, _ := gtxNewScheduler(t, newGtxStore(), 1)
	s.done = make(chan struct{})
	s.signal = make(chan time.Time, 1)

	at := time.Now().Add(time.Minute)
	s.scheduleProcessing(at)
	require.Equal(t, at, <-s.signal)

	s.scheduleIfHaveMoreJobs()
	assert.True(t, (<-s.signal).IsZero())

	// once the scheduler is done the signal is dropped.
	close(s.done)
	s.signal = make(chan time.Time)
	s.scheduleProcessing(at)
	time.Sleep(20 * time.Millisecond)
}

func TestGtxUID(t *testing.T) {
	first, err := UID()
	require.NoError(t, err)
	second, err := UID()
	require.NoError(t, err)

	assert.NotEmpty(t, first)
	assert.NotEqual(t, first, second)
}
