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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestInfExecutor_Register(t *testing.T) {
	h := infHandlerFunc(func(context.Context, string, ProgressReporter) (string, error) { return "", nil })

	t.Run("validations", func(t *testing.T) {
		e := NewExecutor(infNewFakeStore(), infNewFakePubSub())
		require.Error(t, e.Register("", h))
		require.Error(t, e.Register("type", nil))
		require.NoError(t, e.Register("type", h))
		require.Error(t, e.Register("type", h), "duplicate registration")
		e.finishRegistration()
		require.Error(t, e.Register("other", h), "registration complete")
	})
}

func TestInfExecutor_Exec(t *testing.T) {
	ctx := context.Background()

	t.Run("no handler", func(t *testing.T) {
		e := NewExecutor(infNewFakeStore(), infNewFakePubSub())
		_, err := e.exec(ctx, "u", "unknown", "")
		require.ErrorIs(t, err, errNoHandlerDefined)
	})

	t.Run("passes input and returns result", func(t *testing.T) {
		e := NewExecutor(infNewFakeStore(), infNewFakePubSub())
		require.NoError(t, e.Register("t", infHandlerFunc(
			func(_ context.Context, input string, _ ProgressReporter) (string, error) {
				return "got:" + input, nil
			})))
		res, err := e.exec(ctx, "u", "t", "abc")
		require.NoError(t, err)
		require.Equal(t, "got:abc", res)
	})

	t.Run("handler panic is recovered", func(t *testing.T) {
		e := NewExecutor(infNewFakeStore(), infNewFakePubSub())
		require.NoError(t, e.Register("t", infHandlerFunc(
			func(context.Context, string, ProgressReporter) (string, error) { panic("kaboom") })))
		_, err := e.exec(ctx, "u1", "t", "")
		require.ErrorContains(t, err, "panic while processing job=u1 type=t: kaboom")
	})

	t.Run("progress reporter", func(t *testing.T) {
		st := infNewFakeStore(&Job{UID: "u", Type: "t", State: JobStateRunning})
		ps := infNewFakePubSub()
		e := NewExecutor(st, ps)
		var errs []error
		require.NoError(t, e.Register("t", infHandlerFunc(
			func(_ context.Context, _ string, fn ProgressReporter) (string, error) {
				errs = append(errs, fn(-1, ""), fn(101, ""), fn(42, "partial"))
				return "", nil
			})))
		_, err := e.exec(ctx, "u", "t", "")
		require.NoError(t, err)
		require.Error(t, errs[0])
		require.Error(t, errs[1])
		require.NoError(t, errs[2])
		j, _ := st.get("u")
		require.Equal(t, 42, j.RunProgress)
		require.Equal(t, "partial", j.Result)
		require.Equal(t, 1, ps.count(PubSubTopicStateChange))
	})

	t.Run("progress reporter store and publish errors", func(t *testing.T) {
		st := infNewFakeStore(&Job{UID: "u"})
		ps := infNewFakePubSub()
		e := NewExecutor(st, ps)
		var errs []error
		require.NoError(t, e.Register("t", infHandlerFunc(
			func(_ context.Context, _ string, fn ProgressReporter) (string, error) {
				ps.errPublish = errInfBoom
				errs = append(errs, fn(10, "")) // publish failure is only logged
				st.errUpdateProg = errInfBoom
				errs = append(errs, fn(20, ""))
				return "", nil
			})))
		_, err := e.exec(ctx, "u", "t", "")
		require.NoError(t, err)
		require.NoError(t, errs[0])
		require.ErrorIs(t, errs[1], errInfBoom)
	})
}

func TestInfDefinition_Validate(t *testing.T) {
	valid := Definition{UID: "u", Type: "t", Timeout: time.Second}
	tests := []struct {
		name    string
		mutate  func(d *Definition)
		wantErr string
	}{
		{"valid", func(*Definition) {}, ""},
		{"no type", func(d *Definition) { d.Type = "" }, "Type"},
		{"no uid", func(d *Definition) { d.UID = "" }, "unique identifier"},
		{"negative retries", func(d *Definition) { d.MaxRetries = -1 }, "MaxRetries"},
		{"short timeout", func(d *Definition) { d.Timeout = time.Millisecond }, "Timeout"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := valid
			tt.mutate(&d)
			err := d.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestInfDefinition_ToNewJob(t *testing.T) {
	d := Definition{UID: "u", Type: "t", Timeout: 90 * time.Second, MaxRetries: 3, Data: "d"}
	j := d.toNewJob()
	require.Equal(t, "u", j.UID)
	require.Equal(t, "t", j.Type)
	require.Equal(t, "d", j.Data)
	require.Equal(t, 90, j.MaxDurationSeconds)
	require.Equal(t, 3, j.MaxRetries)
	require.Equal(t, JobStateScheduled, j.State)
	require.Equal(t, JobPriorityNormal, j.Priority)
	require.False(t, j.IsRecurring)
	require.Equal(t, j.Created, j.Scheduled)
}

func TestInfState(t *testing.T) {
	require.True(t, JobStateFinished.IsCompleted())
	require.True(t, JobStateFailed.IsCompleted())
	require.True(t, JobStateCanceled.IsCompleted())
	require.False(t, JobStateRunning.IsCompleted())
	require.False(t, JobStateScheduled.IsCompleted())

	require.Len(t, State("").Enum(), 5)

	all, def := GetAllJobStates()
	require.Len(t, all, 5)
	require.Empty(t, def)

	s, ok := State("running").Sanitize()
	require.True(t, ok)
	require.Equal(t, JobStateRunning, s)

	s, ok = State("bogus").Sanitize()
	require.False(t, ok)
	require.Empty(t, s)

	require.Equal(t, Progress{State: JobStateFailed, Progress: ProgressMax}, FailProgress())
}

func TestInfSanitize_WithDefault(t *testing.T) {
	all := func() ([]string, string) { return []string{"a", "b"}, "a" }
	v, ok := Sanitize("", all)
	require.True(t, ok)
	require.Equal(t, "a", v)
	v, ok = Sanitize("b", all)
	require.True(t, ok)
	require.Equal(t, "b", v)
	v, ok = Sanitize("zzz", all)
	require.False(t, ok)
	require.Equal(t, "a", v)
}

func TestInfStateChange_EncodeDecode(t *testing.T) {
	j := &Job{UID: "u", Type: "t", State: JobStateFailed, RunProgress: 55, Result: "r", LastFailureError: "e"}
	payload, err := encodeStateChange(j)
	require.NoError(t, err)
	sc, err := DecodeStateChange(payload)
	require.NoError(t, err)
	require.Equal(t, &StateChange{UID: "u", Type: "t", State: JobStateFailed, Progress: 55, Result: "r", Failure: "e"}, sc)

	_, err = DecodeStateChange([]byte("not gob"))
	require.Error(t, err)

	ps := infNewFakePubSub()
	require.NoError(t, publishStateChange(context.Background(), ps, j))
	require.Equal(t, 1, ps.count(PubSubTopicStateChange))
	ps.errPublish = errInfBoom
	require.ErrorIs(t, publishStateChange(context.Background(), ps, j), errInfBoom)
}

func TestInfGlobalLock(t *testing.T) {
	mx, err := globalLock(context.Background(), &infFakeMutexManager{})
	require.NoError(t, err)
	require.Equal(t, "jobs", mx.Key())

	_, err = globalLock(context.Background(), &infFakeMutexManager{errNew: errInfBoom})
	require.ErrorIs(t, err, errInfBoom)

	_, err = globalLock(context.Background(), &infFakeMutexManager{errLock: errInfBoom})
	require.ErrorIs(t, err, errInfBoom)
}

func TestInfJobPurge_Handle(t *testing.T) {
	ctx := context.Background()

	require.Zero(t, newJobPurge(nil, nil, -time.Hour).minOldAge)

	t.Run("nothing to delete", func(t *testing.T) {
		h := newJobPurge(infNewFakeStore(), &infFakeMutexManager{}, time.Hour)
		res, err := h.Handle(ctx, "", nil)
		require.NoError(t, err)
		require.Equal(t, "no old jobs found", res)
	})
	t.Run("deleted", func(t *testing.T) {
		st := infNewFakeStore()
		st.deletedOld = 4
		h := newJobPurge(st, &infFakeMutexManager{errUnlock: errInfBoom}, time.Hour)
		res, err := h.Handle(ctx, "", nil)
		require.NoError(t, err)
		require.Equal(t, "deleted 4 old jobs", res)
	})
	t.Run("lock error", func(t *testing.T) {
		h := newJobPurge(infNewFakeStore(), &infFakeMutexManager{errLock: errInfBoom}, time.Hour)
		_, err := h.Handle(ctx, "", nil)
		require.Error(t, err)
	})
	t.Run("store error", func(t *testing.T) {
		st := infNewFakeStore()
		st.errDeleteOld = errInfBoom
		h := newJobPurge(st, &infFakeMutexManager{}, time.Hour)
		_, err := h.Handle(ctx, "", nil)
		require.Error(t, err)
	})
}

func TestInfJobOverdue_Handle(t *testing.T) {
	ctx := context.Background()
	past := time.Now().Add(-time.Hour).UnixMilli()
	overdue := func(uid string, retries int) *Job {
		return &Job{UID: uid, State: JobStateRunning, RunDeadline: past, MaxRetries: retries}
	}

	t.Run("no overdue jobs", func(t *testing.T) {
		s, _ := infNewScheduler(infNewFakeStore(), &infFakeMutexManager{}, infNewFakePubSub(), 1)
		h := newJobOverdue(infNewFakeStore(), &infFakeMutexManager{}, s)
		res, err := h.Handle(ctx, "", nil)
		require.NoError(t, err)
		require.Empty(t, res)
	})

	t.Run("fails and reschedules overdue jobs", func(t *testing.T) {
		st := infNewFakeStore(overdue("failed", 0), overdue("retried", 1))
		s, _ := infNewScheduler(st, &infFakeMutexManager{}, infNewFakePubSub(), 1)
		h := newJobOverdue(st, &infFakeMutexManager{errUnlock: errInfBoom}, s)

		res, err := h.Handle(ctx, "", nil)
		require.NoError(t, err)
		require.Equal(t, "found 2 overdue jobs", res)

		j, _ := st.get("failed")
		require.Equal(t, JobStateFailed, j.State)
		require.Equal(t, "deadline exceeded", j.LastFailureError)
		j, _ = st.get("retried")
		require.Equal(t, JobStateScheduled, j.State)

		select {
		case <-s.signal:
		case <-time.After(5 * time.Second):
			t.Fatal("expected a processing signal for the rescheduled job")
		}
	})

	t.Run("errors", func(t *testing.T) {
		s, _ := infNewScheduler(infNewFakeStore(), &infFakeMutexManager{}, infNewFakePubSub(), 1)

		h := newJobOverdue(infNewFakeStore(), &infFakeMutexManager{errLock: errInfBoom}, s)
		_, err := h.Handle(ctx, "", nil)
		require.Error(t, err)

		st := infNewFakeStore()
		st.errListOverdue = errInfBoom
		_, err = newJobOverdue(st, &infFakeMutexManager{}, s).Handle(ctx, "", nil)
		require.Error(t, err)

		st = infNewFakeStore(overdue("x", 0))
		st.errUpdateExec = errors.New("update failed")
		_, err = newJobOverdue(st, &infFakeMutexManager{}, s).Handle(ctx, "", nil)
		require.Error(t, err)
	})
}
