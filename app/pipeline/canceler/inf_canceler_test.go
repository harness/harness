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

package canceler

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/store"
	mocksse "github.com/harness/gitness/mocks/sse"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type infExecutionStore struct {
	store.ExecutionStore

	updated []types.Execution
	err     error
}

func (s *infExecutionStore) Update(_ context.Context, e *types.Execution) error {
	if s.err != nil {
		return s.err
	}
	s.updated = append(s.updated, *e)
	return nil
}

type infStageStore struct {
	store.StageStore

	stages    []*types.Stage
	listErr   error
	updateErr error
	updated   []int64
}

func (s *infStageStore) ListWithSteps(context.Context, int64) ([]*types.Stage, error) {
	return s.stages, s.listErr
}

func (s *infStageStore) Update(_ context.Context, st *types.Stage) error {
	s.updated = append(s.updated, st.ID)
	return s.updateErr
}

type infStepStore struct {
	store.StepStore

	updateErr error
	updated   []int64
}

func (s *infStepStore) Update(_ context.Context, st *types.Step) error {
	s.updated = append(s.updated, st.ID)
	return s.updateErr
}

func TestInfCancel_SkipsCompletedExecutions(t *testing.T) {
	for _, status := range []enum.CIStatus{enum.CIStatusSuccess, enum.CIStatusFailure, enum.CIStatusKilled} {
		es := &infExecutionStore{}
		svc := New(es, &mocksse.Streamer{}, nil, nil, &infStageStore{}, &infStepStore{})
		require.NoError(t, svc.Cancel(context.Background(), &types.RepositoryCore{}, &types.Execution{Status: status}))
		require.Empty(t, es.updated, status)
	}
}

func TestInfCancel_KillsRunningAndSkipsPendingWork(t *testing.T) {
	stages := []*types.Stage{
		{ID: 1, Number: 1, Status: enum.CIStatusSuccess}, // already done: untouched
		{
			ID: 2, Number: 2, Status: enum.CIStatusRunning, Started: 5,
			Steps: []*types.Step{
				{ID: 20, Number: 1, Status: enum.CIStatusSuccess},
				{ID: 21, Number: 2, Status: enum.CIStatusRunning, Started: 6},
				{ID: 22, Number: 3, Status: enum.CIStatusPending},
			},
		},
		{ID: 3, Number: 3, Status: enum.CIStatusPending},
	}
	es := &infExecutionStore{}
	ss := &infStageStore{stages: stages}
	steps := &infStepStore{}
	streamer := &mocksse.Streamer{}
	streamer.On("Publish", int64(9), enum.SSETypeExecutionCanceled, mock.Anything).Return()

	svc := New(es, streamer, nil, nil, ss, steps)
	execution := &types.Execution{ID: 4, Status: enum.CIStatusPending}
	require.NoError(t, svc.Cancel(context.Background(), &types.RepositoryCore{ParentID: 9}, execution))

	require.Equal(t, enum.CIStatusKilled, execution.Status)
	require.NotZero(t, execution.Finished)
	require.Equal(t, execution.Finished, execution.Started, "never-started executions get a start time")
	require.Len(t, es.updated, 1)
	require.Len(t, execution.Stages, 3)

	require.Equal(t, enum.CIStatusSuccess, stages[0].Status)
	require.Equal(t, enum.CIStatusKilled, stages[1].Status)
	require.Equal(t, enum.CIStatusSkipped, stages[2].Status)
	require.NotZero(t, stages[2].Started)
	require.NotZero(t, stages[1].Stopped)
	require.Equal(t, []int64{2, 3}, ss.updated)

	require.Equal(t, enum.CIStatusSuccess, stages[1].Steps[0].Status)
	require.Equal(t, enum.CIStatusKilled, stages[1].Steps[1].Status)
	require.Equal(t, enum.CIStatusSkipped, stages[1].Steps[2].Status)
	require.Equal(t, 130, stages[1].Steps[1].ExitCode)
	require.Equal(t, []int64{21, 22}, steps.updated)

	streamer.AssertExpectations(t)
}

func TestInfCancel_KeepsExistingStartTime(t *testing.T) {
	streamer := &mocksse.Streamer{}
	streamer.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return()
	svc := New(&infExecutionStore{}, streamer, nil, nil, &infStageStore{}, &infStepStore{})
	execution := &types.Execution{Status: enum.CIStatusRunning, Started: 123}
	require.NoError(t, svc.Cancel(context.Background(), &types.RepositoryCore{}, execution))
	require.EqualValues(t, 123, execution.Started)
}

func TestInfCancel_StoreFailures(t *testing.T) {
	ctx := context.Background()
	repo := &types.RepositoryCore{}

	t.Run("execution update fails", func(t *testing.T) {
		svc := New(&infExecutionStore{err: errors.New("locked")}, &mocksse.Streamer{}, nil, nil,
			&infStageStore{}, &infStepStore{})
		err := svc.Cancel(ctx, repo, &types.Execution{Status: enum.CIStatusRunning})
		require.ErrorContains(t, err, "could not update execution status to canceled")
	})
	t.Run("stage listing fails", func(t *testing.T) {
		svc := New(&infExecutionStore{}, &mocksse.Streamer{}, nil, nil,
			&infStageStore{listErr: errors.New("db")}, &infStepStore{})
		err := svc.Cancel(ctx, repo, &types.Execution{Status: enum.CIStatusRunning})
		require.ErrorContains(t, err, "could not list stages with steps")
	})
	t.Run("stage and step update failures are only logged", func(t *testing.T) {
		stages := []*types.Stage{{
			ID: 1, Status: enum.CIStatusRunning,
			Steps: []*types.Step{{ID: 2, Status: enum.CIStatusRunning}},
		}}
		streamer := &mocksse.Streamer{}
		streamer.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return()
		ss := &infStageStore{stages: stages, updateErr: errors.New("conflict")}
		steps := &infStepStore{updateErr: errors.New("conflict")}
		svc := New(&infExecutionStore{}, streamer, nil, nil, ss, steps)
		require.NoError(t, svc.Cancel(ctx, repo, &types.Execution{Status: enum.CIStatusRunning}))
		require.Equal(t, []int64{1}, ss.updated)
		require.Equal(t, []int64{2}, steps.updated)
		streamer.AssertCalled(t, "Publish", mock.Anything, enum.SSETypeExecutionCanceled, mock.Anything)
	})
}
