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
	"sync/atomic"
	"testing"
	"time"

	"github.com/harness/gitness/lock"

	"github.com/stretchr/testify/require"
)

func infScheduledJob(uid string) *Job {
	now := time.Now().UnixMilli()
	return &Job{
		UID:                uid,
		Type:               "inf-type",
		State:              JobStateScheduled,
		Scheduled:          now - 1000,
		MaxDurationSeconds: 5,
		Created:            now,
		Updated:            now,
	}
}

func TestInfScheduler_NewSchedulerClampsMaxRunning(t *testing.T) {
	s, err := NewScheduler(nil, nil, nil, nil, "id", 0, time.Hour)
	require.NoError(t, err)
	require.Equal(t, 1, s.maxRunning)
	s, err = NewScheduler(nil, nil, nil, nil, "id", 7, time.Hour)
	require.NoError(t, err)
	require.Equal(t, 7, s.maxRunning)
}

func TestInfScheduler_ProcessReadyJobs_Success(t *testing.T) {
	st := infNewFakeStore(infScheduledJob("j1"))
	ps := infNewFakePubSub()
	s, exec := infNewScheduler(st, &infFakeMutexManager{}, ps, 5)

	var gotInput atomic.Value
	require.NoError(t, exec.Register("inf-type", infHandlerFunc(
		func(_ context.Context, input string, fn ProgressReporter) (string, error) {
			gotInput.Store(input)
			require.NoError(t, fn(50, "half"))
			return "done", nil
		})))
	exec.finishRegistration()

	count, next, gotAll, err := s.processReadyJobs(context.Background(), time.Now())
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.True(t, gotAll)
	require.True(t, next.IsZero())

	s.WaitJobsDone(context.Background())

	j, ok := st.get("j1")
	require.True(t, ok)
	require.Equal(t, JobStateFinished, j.State)
	require.Equal(t, ProgressMax, j.RunProgress)
	require.Equal(t, "done", j.Result)
	require.Equal(t, 1, j.TotalExecutions)
	require.Empty(t, j.RunBy)
	require.GreaterOrEqual(t, ps.count(PubSubTopicStateChange), 3) // start, progress, finish
}

func TestInfScheduler_ProcessReadyJobs_FailureAndRetry(t *testing.T) {
	failing := infScheduledJob("retry")
	failing.MaxRetries = 1
	noRetry := infScheduledJob("noretry")
	st := infNewFakeStore(failing, noRetry)
	s, exec := infNewScheduler(st, &infFakeMutexManager{}, infNewFakePubSub(), 5)
	require.NoError(t, exec.Register("inf-type", infHandlerFunc(
		func(context.Context, string, ProgressReporter) (string, error) {
			return "", errors.New("handler failed")
		})))

	count, _, _, err := s.processReadyJobs(context.Background(), time.Now())
	require.NoError(t, err)
	require.Equal(t, 2, count)
	s.WaitJobsDone(context.Background())

	j, _ := st.get("retry")
	require.Equal(t, JobStateScheduled, j.State)
	require.Equal(t, 1, j.ConsecutiveFailures)
	require.Equal(t, "handler failed", j.LastFailureError)
	require.Greater(t, j.Scheduled, time.Now().UnixMilli())

	j, _ = st.get("noretry")
	require.Equal(t, JobStateFailed, j.State)
	require.Equal(t, "handler failed", j.LastFailureError)
}

func TestInfScheduler_ProcessReadyJobs_UnknownHandlerFails(t *testing.T) {
	st := infNewFakeStore(infScheduledJob("j1"))
	s, _ := infNewScheduler(st, &infFakeMutexManager{}, infNewFakePubSub(), 5)

	_, _, _, err := s.processReadyJobs(context.Background(), time.Now())
	require.NoError(t, err)
	s.WaitJobsDone(context.Background())

	j, _ := st.get("j1")
	require.Equal(t, JobStateFailed, j.State)
	require.Contains(t, j.LastFailureError, errNoHandlerDefined.Error())
}

func TestInfScheduler_ProcessReadyJobs_Capacity(t *testing.T) {
	st := infNewFakeStore(infScheduledJob("a"), infScheduledJob("b"))
	s, exec := infNewScheduler(st, &infFakeMutexManager{}, infNewFakePubSub(), 1)
	release := make(chan struct{})
	require.NoError(t, exec.Register("inf-type", infHandlerFunc(
		func(context.Context, string, ProgressReporter) (string, error) {
			<-release
			return "", nil
		})))

	count, _, gotAll, err := s.processReadyJobs(context.Background(), time.Now())
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.False(t, gotAll)

	// no slot left while the first job is running
	count, _, gotAll, err = s.processReadyJobs(context.Background(), time.Now())
	require.NoError(t, err)
	require.Equal(t, 0, count)
	require.False(t, gotAll)

	close(release)
	s.WaitJobsDone(context.Background())
}

func TestInfScheduler_ProcessReadyJobs_KnownNextTime(t *testing.T) {
	future := infScheduledJob("future")
	future.Scheduled = time.Now().Add(time.Hour).UnixMilli()
	st := infNewFakeStore(future)
	s, _ := infNewScheduler(st, &infFakeMutexManager{}, infNewFakePubSub(), 5)

	count, next, gotAll, err := s.processReadyJobs(context.Background(), time.Now())
	require.NoError(t, err)
	require.Zero(t, count)
	require.True(t, gotAll)
	require.WithinDuration(t, time.UnixMilli(future.Scheduled), next, time.Millisecond)
}

func TestInfScheduler_ProcessReadyJobs_Errors(t *testing.T) {
	tests := []struct {
		name  string
		setup func(st *infFakeStore, mm *infFakeMutexManager)
		want  string
	}{
		{"lock new", func(_ *infFakeStore, mm *infFakeMutexManager) { mm.errNew = errInfBoom }, "global lock"},
		{"lock", func(_ *infFakeStore, mm *infFakeMutexManager) { mm.errLock = errInfBoom }, "global lock"},
		{"count running", func(st *infFakeStore, _ *infFakeMutexManager) { st.errCountRun = errInfBoom }, "available slots"},
		{"list ready", func(st *infFakeStore, _ *infFakeMutexManager) { st.errListReady = errInfBoom }, "load scheduled"},
		{"next time", func(st *infFakeStore, _ *infFakeMutexManager) { st.errNextTime = errInfBoom }, "next scheduled"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := infNewFakeStore()
			mm := &infFakeMutexManager{}
			tt.setup(st, mm)
			s, _ := infNewScheduler(st, mm, infNewFakePubSub(), 5)
			_, _, gotAll, err := s.processReadyJobs(context.Background(), time.Now())
			require.ErrorIs(t, err, errInfBoom)
			require.Contains(t, err.Error(), tt.want)
			require.False(t, gotAll)
		})
	}
}

func TestInfScheduler_ProcessReadyJobs_UnlockAndPublishErrorsAreTolerated(t *testing.T) {
	st := infNewFakeStore(infScheduledJob("j1"))
	ps := infNewFakePubSub()
	ps.errPublish = errInfBoom
	s, exec := infNewScheduler(st, &infFakeMutexManager{errUnlock: errInfBoom}, ps, 5)
	require.NoError(t, exec.Register("inf-type", infHandlerFunc(
		func(context.Context, string, ProgressReporter) (string, error) { return "ok", nil })))

	count, _, _, err := s.processReadyJobs(context.Background(), time.Now())
	require.NoError(t, err)
	require.Equal(t, 1, count)
	s.WaitJobsDone(context.Background())
	j, _ := st.get("j1")
	require.Equal(t, JobStateFinished, j.State)
}

func TestInfScheduler_ProcessReadyJobs_UpdateExecutionError(t *testing.T) {
	st := infNewFakeStore(infScheduledJob("j1"))
	st.errUpdateExec = errInfBoom
	s, _ := infNewScheduler(st, &infFakeMutexManager{}, infNewFakePubSub(), 5)

	count, next, gotAll, err := s.processReadyJobs(context.Background(), time.Now())
	require.NoError(t, err)
	require.Zero(t, count)
	require.False(t, gotAll)
	require.True(t, next.IsZero())
}

func TestInfScheduler_RunJob(t *testing.T) {
	st := infNewFakeStore()
	s, _ := infNewScheduler(st, &infFakeMutexManager{}, infNewFakePubSub(), 5)

	require.Error(t, s.RunJob(context.Background(), Definition{}))

	def := Definition{UID: "u1", Type: "t", Timeout: 3 * time.Second, Data: "payload", MaxRetries: 2}
	require.NoError(t, s.RunJob(context.Background(), def))
	j, ok := st.get("u1")
	require.True(t, ok)
	require.Equal(t, JobStateScheduled, j.State)
	require.Equal(t, "payload", j.Data)
	require.Equal(t, 3, j.MaxDurationSeconds)

	select {
	case <-s.signal:
	case <-time.After(5 * time.Second):
		t.Fatal("expected scheduling signal")
	}

	st.errCreate = errInfBoom
	err := s.RunJob(context.Background(), Definition{UID: "u2", Type: "t", Timeout: time.Second})
	require.ErrorIs(t, err, errInfBoom)
}

func TestInfScheduler_RunJobs(t *testing.T) {
	st := infNewFakeStore()
	s, _ := infNewScheduler(st, &infFakeMutexManager{}, infNewFakePubSub(), 5)

	require.NoError(t, s.RunJobs(context.Background(), "grp", nil))

	err := s.RunJobs(context.Background(), "grp", []Definition{{UID: "ok", Type: "t", Timeout: time.Second}, {}})
	require.Error(t, err)

	defs := []Definition{
		{UID: "a", Type: "t", Timeout: time.Second},
		{UID: "b", Type: "t", Timeout: time.Second},
	}
	require.NoError(t, s.RunJobs(context.Background(), "grp", defs))
	list, err := st.ListByGroupID(context.Background(), "grp")
	require.NoError(t, err)
	require.Len(t, list, 2)

	st.errCreate = errInfBoom
	err = s.RunJobs(context.Background(), "grp", []Definition{{UID: "c", Type: "t", Timeout: time.Second}})
	require.ErrorIs(t, err, errInfBoom)
}

func TestInfScheduler_CancelJob(t *testing.T) {
	recurring := infScheduledJob("recurring")
	recurring.IsRecurring = true
	finished := infScheduledJob("finished")
	finished.State = JobStateFinished
	scheduled := infScheduledJob("scheduled")
	running := infScheduledJob("running")
	running.State = JobStateRunning

	t.Run("not found is success", func(t *testing.T) {
		s, _ := infNewScheduler(infNewFakeStore(), &infFakeMutexManager{}, infNewFakePubSub(), 1)
		require.NoError(t, s.CancelJob(context.Background(), "missing"))
	})
	t.Run("lock error", func(t *testing.T) {
		s, _ := infNewScheduler(infNewFakeStore(), &infFakeMutexManager{errLock: errInfBoom}, infNewFakePubSub(), 1)
		require.ErrorIs(t, s.CancelJob(context.Background(), "x"), errInfBoom)
	})
	t.Run("find error", func(t *testing.T) {
		st := infNewFakeStore()
		st.errFind = errInfBoom
		s, _ := infNewScheduler(st, &infFakeMutexManager{}, infNewFakePubSub(), 1)
		require.ErrorIs(t, s.CancelJob(context.Background(), "x"), errInfBoom)
	})
	t.Run("recurring rejected", func(t *testing.T) {
		s, _ := infNewScheduler(infNewFakeStore(recurring), &infFakeMutexManager{}, infNewFakePubSub(), 1)
		require.Error(t, s.CancelJob(context.Background(), "recurring"))
	})
	t.Run("finished is noop", func(t *testing.T) {
		st := infNewFakeStore(finished)
		s, _ := infNewScheduler(st, &infFakeMutexManager{}, infNewFakePubSub(), 1)
		require.NoError(t, s.CancelJob(context.Background(), "finished"))
		j, _ := st.get("finished")
		require.Equal(t, JobStateFinished, j.State)
	})
	t.Run("update error", func(t *testing.T) {
		st := infNewFakeStore(scheduled)
		st.errUpdateExec = errInfBoom
		s, _ := infNewScheduler(st, &infFakeMutexManager{}, infNewFakePubSub(), 1)
		require.ErrorIs(t, s.CancelJob(context.Background(), "scheduled"), errInfBoom)
	})
	t.Run("scheduled job published to other instances", func(t *testing.T) {
		st := infNewFakeStore(scheduled)
		ps := infNewFakePubSub()
		s, _ := infNewScheduler(st, &infFakeMutexManager{errUnlock: errInfBoom}, ps, 1)
		require.NoError(t, s.CancelJob(context.Background(), "scheduled"))
		j, _ := st.get("scheduled")
		require.Equal(t, JobStateCanceled, j.State)
		require.Equal(t, 1, ps.count(PubSubTopicCancelJob))
	})
	t.Run("locally running job is canceled directly", func(t *testing.T) {
		st := infNewFakeStore(running)
		ps := infNewFakePubSub()
		s, _ := infNewScheduler(st, &infFakeMutexManager{}, ps, 1)
		canceled := false
		s.cancelJobMap["running"] = func() { canceled = true }
		require.NoError(t, s.CancelJob(context.Background(), "running"))
		require.True(t, canceled)
		require.Zero(t, ps.count(PubSubTopicCancelJob))
	})
}

func TestInfScheduler_HandleCancelJob(t *testing.T) {
	s, _ := infNewScheduler(infNewFakeStore(), &infFakeMutexManager{}, infNewFakePubSub(), 1)
	called := 0
	s.cancelJobMap["known"] = func() { called++ }

	require.NoError(t, s.handleCancelJob(nil))
	require.NoError(t, s.handleCancelJob([]byte("unknown")))
	require.Zero(t, called)
	require.NoError(t, s.handleCancelJob([]byte("known")))
	require.Equal(t, 1, called)
}

func TestInfScheduler_DoExecAlreadyRunning(t *testing.T) {
	s, _ := infNewScheduler(infNewFakeStore(), &infFakeMutexManager{}, infNewFakePubSub(), 1)
	s.cancelJobMap["dup"] = func() {}
	res, failure := s.doExec(context.Background(), "dup", "t", "", time.Now().Add(time.Minute).UnixMilli())
	require.Empty(t, res)
	require.Contains(t, failure, "already running")
}

func TestInfScheduler_DoExecDeadline(t *testing.T) {
	s, exec := infNewScheduler(infNewFakeStore(), &infFakeMutexManager{}, infNewFakePubSub(), 1)
	require.NoError(t, exec.Register("t", infHandlerFunc(
		func(ctx context.Context, _ string, _ ProgressReporter) (string, error) {
			<-ctx.Done()
			return "", ctx.Err()
		})))
	_, failure := s.doExec(context.Background(), "u", "t", "", time.Now().Add(20*time.Millisecond).UnixMilli())
	require.Contains(t, failure, context.DeadlineExceeded.Error())
	require.Empty(t, s.cancelJobMap)
}

func TestInfScheduler_PreExec(t *testing.T) {
	s, _ := infNewScheduler(infNewFakeStore(), &infFakeMutexManager{}, infNewFakePubSub(), 1)
	j := &Job{MaxDurationSeconds: 0, TotalExecutions: 2, Result: "old", LastFailureError: "old"}
	s.preExec(j)
	require.Equal(t, 1, j.MaxDurationSeconds)
	require.Equal(t, JobStateRunning, j.State)
	require.Equal(t, "inst-1", j.RunBy)
	require.Equal(t, 3, j.TotalExecutions)
	require.Empty(t, j.Result)
	require.Empty(t, j.LastFailureError)
	require.Greater(t, j.RunDeadline, time.Now().UnixMilli())
}

func TestInfPostExec(t *testing.T) {
	tests := []struct {
		name      string
		job       Job
		result    string
		failure   string
		wantState State
		check     func(t *testing.T, j *Job)
	}{
		{
			name:      "scheduled job is untouched",
			job:       Job{State: JobStateScheduled},
			wantState: JobStateScheduled,
		},
		{
			name:      "success",
			job:       Job{State: JobStateRunning, RunBy: "x"},
			result:    "res",
			wantState: JobStateFinished,
			check: func(t *testing.T, j *Job) {
				require.Equal(t, ProgressMax, j.RunProgress)
				require.Equal(t, "res", j.Result)
				require.Empty(t, j.RunBy)
			},
		},
		{
			name:      "failure without retries",
			job:       Job{State: JobStateRunning},
			failure:   "bad",
			wantState: JobStateFailed,
			check: func(t *testing.T, j *Job) {
				require.Equal(t, 1, j.ConsecutiveFailures)
				require.Equal(t, "bad", j.LastFailureError)
			},
		},
		{
			name:      "failure with retry rescheduled",
			job:       Job{State: JobStateRunning, MaxRetries: 2, RunProgress: 30},
			failure:   "bad",
			wantState: JobStateScheduled,
			check: func(t *testing.T, j *Job) {
				require.Zero(t, j.RunProgress)
				require.Greater(t, j.Scheduled, time.Now().UnixMilli())
			},
		},
		{
			name:      "canceled but succeeded is finished",
			job:       Job{State: JobStateCanceled},
			wantState: JobStateFinished,
		},
		{
			name:      "canceled and failed stays canceled",
			job:       Job{State: JobStateCanceled},
			failure:   "bad",
			wantState: JobStateCanceled,
		},
		{
			name:      "recurring success is rescheduled",
			job:       Job{State: JobStateRunning, IsRecurring: true, RecurringCron: "* * * * *", ConsecutiveFailures: 3},
			wantState: JobStateScheduled,
			check: func(t *testing.T, j *Job) {
				require.Zero(t, j.ConsecutiveFailures)
				require.Greater(t, j.Scheduled, time.Now().UnixMilli())
			},
		},
		{
			name:      "recurring failure is rescheduled keeping failure error",
			job:       Job{State: JobStateRunning, IsRecurring: true, RecurringCron: "* * * * *"},
			failure:   "bad",
			wantState: JobStateScheduled,
			check: func(t *testing.T, j *Job) {
				require.Equal(t, 1, j.ConsecutiveFailures)
				require.Equal(t, "bad", j.LastFailureError)
			},
		},
		{
			name:      "recurring with invalid cron fails",
			job:       Job{State: JobStateRunning, IsRecurring: true, RecurringCron: "nope"},
			failure:   "bad",
			wantState: JobStateFailed,
			check: func(t *testing.T, j *Job) {
				require.Contains(t, j.LastFailureError, "failed to parse cron string")
				require.Contains(t, j.LastFailureError, "bad")
			},
		},
		{
			name:      "recurring with invalid cron and no failure",
			job:       Job{State: JobStateRunning, IsRecurring: true, RecurringCron: "nope"},
			wantState: JobStateFailed,
			check: func(t *testing.T, j *Job) {
				require.Contains(t, j.LastFailureError, "failed to parse cron string")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := tt.job
			postExec(&j, tt.result, tt.failure)
			require.Equal(t, tt.wantState, j.State)
			if tt.check != nil {
				tt.check(t, &j)
			}
		})
	}
}

func TestInfScheduler_ProgressAndPurge(t *testing.T) {
	a := infScheduledJob("a")
	a.GroupID = "g"
	a.RunProgress = 40
	a.Result = "r"
	a.LastFailureError = "f"
	b := infScheduledJob("b")
	b.GroupID = "g"
	st := infNewFakeStore(a, b)
	s, _ := infNewScheduler(st, &infFakeMutexManager{}, infNewFakePubSub(), 1)
	ctx := context.Background()

	p, err := s.GetJobProgress(ctx, "a")
	require.NoError(t, err)
	require.Equal(t, Progress{State: JobStateScheduled, Progress: 40, Result: "r", Failure: "f"}, p)

	_, err = s.GetJobProgress(ctx, "missing")
	require.Error(t, err)

	ps, err := s.GetJobProgressForGroup(ctx, "g")
	require.NoError(t, err)
	require.Len(t, ps, 2)

	ps, err = s.GetJobProgressForGroup(ctx, "none")
	require.NoError(t, err)
	require.Empty(t, ps)

	st.errListGroup = errInfBoom
	_, err = s.GetJobProgressForGroup(ctx, "g")
	require.ErrorIs(t, err, errInfBoom)
	st.errListGroup = nil

	n, err := s.PurgeJobsByGroupID(ctx, "g")
	require.NoError(t, err)
	require.EqualValues(t, 2, n)

	st.errDeleteGroup = errInfBoom
	_, err = s.PurgeJobsByGroupID(ctx, "g")
	require.ErrorIs(t, err, errInfBoom)

	require.NoError(t, s.PurgeJobByUID(ctx, "a"))
	st.errDeleteUID = errInfBoom
	require.ErrorIs(t, s.PurgeJobByUID(ctx, "a"), errInfBoom)

	require.Nil(t, mapToProgressMany(nil))
}

func TestInfScheduler_AddRecurring(t *testing.T) {
	st := infNewFakeStore()
	s, _ := infNewScheduler(st, &infFakeMutexManager{}, infNewFakePubSub(), 1)
	ctx := context.Background()

	require.Error(t, s.AddRecurring(ctx, "u", "t", "bad cron", time.Second))

	require.NoError(t, s.AddRecurring(ctx, "u", "t", "*/5 * * * *", 10*time.Second))
	j, ok := st.get("u")
	require.True(t, ok)
	require.True(t, j.IsRecurring)
	require.Equal(t, 10, j.MaxDurationSeconds)
	require.Equal(t, JobPriorityElevated, j.Priority)
	require.Greater(t, j.Scheduled, time.Now().UnixMilli())

	st.errUpsert = errInfBoom
	require.ErrorIs(t, s.AddRecurring(ctx, "u", "t", "*/5 * * * *", time.Second), errInfBoom)
}

func TestInfScheduler_CreateAndRegisterNecessaryJobs(t *testing.T) {
	st := infNewFakeStore()
	s, exec := infNewScheduler(st, &infFakeMutexManager{}, infNewFakePubSub(), 1)
	ctx := context.Background()

	require.NoError(t, s.createNecessaryJobs(ctx))
	for _, uid := range []string{jobUIDPurge, jobUIDOverdue} {
		j, ok := st.get(uid)
		require.True(t, ok)
		require.True(t, j.IsRecurring)
	}
	require.NoError(t, s.registerNecessaryJobs())
	require.Contains(t, exec.handlerMap, jobTypePurge)
	require.Contains(t, exec.handlerMap, jobTypeOverdue)
	require.Error(t, s.registerNecessaryJobs(), "double registration must fail")

	t.Run("lock error", func(t *testing.T) {
		s2, _ := infNewScheduler(infNewFakeStore(), &infFakeMutexManager{errLock: errInfBoom}, infNewFakePubSub(), 1)
		require.ErrorIs(t, s2.createNecessaryJobs(ctx), errInfBoom)
	})
	t.Run("upsert error", func(t *testing.T) {
		st2 := infNewFakeStore()
		st2.errUpsert = errInfBoom
		s2, _ := infNewScheduler(st2, &infFakeMutexManager{errUnlock: errInfBoom}, infNewFakePubSub(), 1)
		require.ErrorIs(t, s2.createNecessaryJobs(ctx), errInfBoom)
	})
	t.Run("pre-registered overdue", func(t *testing.T) {
		s2, exec2 := infNewScheduler(infNewFakeStore(), &infFakeMutexManager{}, infNewFakePubSub(), 1)
		require.NoError(t, exec2.Register(jobTypeOverdue, infHandlerFunc(nil)))
		require.Error(t, s2.registerNecessaryJobs())
	})
	t.Run("pre-registered purge", func(t *testing.T) {
		s2, exec2 := infNewScheduler(infNewFakeStore(), &infFakeMutexManager{}, infNewFakePubSub(), 1)
		require.NoError(t, exec2.Register(jobTypePurge, infHandlerFunc(nil)))
		require.Error(t, s2.registerNecessaryJobs())
	})
}

func TestInfScheduler_Run(t *testing.T) {
	st := infNewFakeStore(infScheduledJob("j1"))
	ps := infNewFakePubSub()
	exec := NewExecutor(st, ps)
	s, err := NewScheduler(st, exec, &infFakeMutexManager{}, ps, "inst", 3, time.Hour)
	require.NoError(t, err)
	require.NoError(t, exec.Register("inf-type", infHandlerFunc(
		func(context.Context, string, ProgressReporter) (string, error) { return "ok", nil })))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() { errCh <- s.Run(ctx) }()

	require.Eventually(t, func() bool {
		j, ok := st.get("j1")
		return ok && j.State == JobStateFinished
	}, 10*time.Second, 10*time.Millisecond)

	cancel()
	select {
	case runErr := <-errCh:
		require.ErrorIs(t, runErr, context.Canceled)
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return")
	}
	s.WaitJobsDone(context.Background())

	require.Equal(t, 1, ps.closedCount)
	require.Equal(t, []string{PubSubTopicCancelJob}, ps.subscribed)
	require.EqualError(t, s.Run(context.Background()), "already started")

	// WaitJobsDone honours context cancellation while jobs are still running.
	s.wgRunning.Add(1)
	canceledCtx, cancelNow := context.WithCancel(context.Background())
	cancelNow()
	s.WaitJobsDone(canceledCtx)
	s.wgRunning.Done()
}

func TestInfScheduler_RunStartupErrors(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name  string
		setup func(st *infFakeStore, mm *infFakeMutexManager, exec *Executor)
		want  string
	}{
		{
			name:  "create necessary jobs",
			setup: func(st *infFakeStore, _ *infFakeMutexManager, _ *Executor) { st.errUpsert = errInfBoom },
			want:  "failed to create necessary jobs",
		},
		{
			name: "register necessary jobs",
			setup: func(_ *infFakeStore, _ *infFakeMutexManager, exec *Executor) {
				_ = exec.Register(jobTypePurge, infHandlerFunc(nil))
			},
			want: "failed to register scheduler's internal jobs",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := infNewFakeStore()
			mm := &infFakeMutexManager{}
			ps := infNewFakePubSub()
			exec := NewExecutor(st, ps)
			s, err := NewScheduler(st, exec, mm, ps, "inst", 1, time.Hour)
			require.NoError(t, err)
			tt.setup(st, mm, exec)
			require.ErrorContains(t, s.Run(ctx), tt.want)
		})
	}
}

var _ lock.MutexManager = (*infFakeMutexManager)(nil)
