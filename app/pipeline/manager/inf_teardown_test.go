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

package manager

import (
	"context"
	"strings"
	"testing"

	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// infWireTeardown prepares the harness so that stage 1 belongs to execution 2.
func infWireTeardown(h *infManagerHarness, execution *types.Execution, stages []*types.Stage) {
	h.executions.findFn = func(int64) (*types.Execution, error) { return execution, nil }
	h.stages.listWithStepsFn = func(int64) ([]*types.Stage, error) { return stages, nil }
	h.stages.findFn = func(id int64) (*types.Stage, error) {
		for _, s := range stages {
			if s.ID == id {
				cp := *s
				cp.Status = enum.CIStatusSkipped
				return &cp, nil
			}
		}
		return nil, errInfBoom
	}
}

func TestInfTeardown_SchedulesDownstreamStages(t *testing.T) {
	ctx := context.Background()
	h := infNewManagerHarness()
	execution := &types.Execution{ID: 2, RepoID: 3, PipelineID: 5, Status: enum.CIStatusRunning}
	first := &types.Stage{ID: 1, ExecutionID: 2, Name: "first", Status: enum.CIStatusSuccess}
	second := &types.Stage{
		ID: 2, ExecutionID: 2, Name: "second", Status: enum.CIStatusWaitingOnDeps,
		DependsOn: []string{"first"}, OnSuccess: true,
	}
	infWireTeardown(h, execution, []*types.Stage{first, second})

	require.NoError(t, h.mgr.AfterStage(ctx, first))

	require.Equal(t, enum.CIStatusPending, second.Status)
	require.Equal(t, []*types.Stage{second}, h.sched.scheduled)
	require.Equal(t, enum.CIStatusRunning, execution.Status, "execution waits for the remaining stage")
	h.sse.AssertNotCalled(t, "Publish", mock.Anything, enum.SSETypeExecutionCompleted, mock.Anything)
}

func TestInfTeardown_CompletesExecution(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name     string
		statuses []enum.CIStatus
		want     enum.CIStatus
	}{
		{"success", []enum.CIStatus{enum.CIStatusSuccess, enum.CIStatusSkipped}, enum.CIStatusSuccess},
		{"failure", []enum.CIStatus{enum.CIStatusSuccess, enum.CIStatusFailure}, enum.CIStatusFailure},
		{"error", []enum.CIStatus{enum.CIStatusSuccess, enum.CIStatusError}, enum.CIStatusError},
		{"killed", []enum.CIStatus{enum.CIStatusKilled, enum.CIStatusFailure}, enum.CIStatusKilled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := infNewManagerHarness()
			execution := &types.Execution{ID: 2, RepoID: 3, PipelineID: 5, Status: enum.CIStatusRunning}
			var stages []*types.Stage
			for i, status := range tt.statuses {
				stages = append(stages, &types.Stage{ID: int64(i + 1), ExecutionID: 2, Status: status})
			}
			infWireTeardown(h, execution, stages)
			stage := &types.Stage{
				ID: 1, ExecutionID: 2, Error: strings.Repeat("x", 600),
				Steps: []*types.Step{{ID: 10, Error: strings.Repeat("y", 600)}, {ID: 11}},
			}

			require.NoError(t, h.mgr.AfterStage(ctx, stage))

			require.Equal(t, tt.want, execution.Status)
			require.NotZero(t, execution.Finished)
			require.Equal(t, execution.Finished, execution.Started, "never-started executions get a start time")
			require.Len(t, stage.Error, 500)
			require.Len(t, h.steps.updated, 2)
			require.Len(t, h.steps.updated[0].Error, 500)
			require.Equal(t, []int64{10, 11}, h.logz.deleted)
			require.Len(t, execution.Stages, len(stages))
			require.Len(t, h.checks.upserted, 1)
			h.sse.AssertCalled(t, "Publish", int64(2), enum.SSETypeExecutionCompleted, mock.Anything)
		})
	}
}

func TestInfTeardown_SkipsDownstreamStagesThatDoNotRunInThisState(t *testing.T) {
	ctx := context.Background()
	h := infNewManagerHarness()
	execution := &types.Execution{ID: 2, RepoID: 3, PipelineID: 5, Status: enum.CIStatusRunning}
	failed := &types.Stage{ID: 1, ExecutionID: 2, Name: "first", Status: enum.CIStatusFailure}
	onSuccessOnly := &types.Stage{
		ID: 2, ExecutionID: 2, Name: "deploy", Status: enum.CIStatusWaitingOnDeps,
		DependsOn: []string{"first"}, OnSuccess: true,
	}
	onFailure := &types.Stage{
		ID: 3, ExecutionID: 2, Name: "notify", Status: enum.CIStatusWaitingOnDeps,
		DependsOn: []string{"first"}, OnFailure: true,
	}
	infWireTeardown(h, execution, []*types.Stage{failed, onSuccessOnly, onFailure})

	require.NoError(t, h.mgr.AfterStage(ctx, failed))

	require.Equal(t, enum.CIStatusSkipped, onSuccessOnly.Status)
	require.NotZero(t, onSuccessOnly.Started)
	require.Equal(t, enum.CIStatusPending, onFailure.Status)
	require.Equal(t, []*types.Stage{onFailure}, h.sched.scheduled)
}

func TestInfTeardown_SkipsSuccessOnlyOnFailureOnlyStagesAfterSuccess(t *testing.T) {
	h := infNewManagerHarness()
	succeeded := &types.Stage{ID: 1, Name: "a", Status: enum.CIStatusSuccess}
	failureOnly := &types.Stage{
		ID: 2, Name: "b", Status: enum.CIStatusWaitingOnDeps, DependsOn: []string{"a"}, OnFailure: true,
	}
	td := &teardown{Stages: h.stages}
	require.NoError(t, td.cancelDownstream(context.Background(), []*types.Stage{succeeded, failureOnly}))
	require.Equal(t, enum.CIStatusSkipped, failureOnly.Status)
}

func TestInfTeardown_VersionConflictResyncsStage(t *testing.T) {
	ctx := context.Background()
	h := infNewManagerHarness()
	h.stages.updateFn = func(*types.Stage) error { return gitnessstore.ErrVersionConflict }
	h.stages.findFn = func(int64) (*types.Stage, error) {
		return &types.Stage{
			Status: enum.CIStatusRunning, Error: "e", ExitCode: 4, Machine: "m", Started: 5, Stopped: 6,
		}, nil
	}
	td := &teardown{Stages: h.stages, Scheduler: h.sched}

	waiting := &types.Stage{
		ID: 1, Name: "w", Status: enum.CIStatusWaitingOnDeps, DependsOn: []string{"a"}, OnSuccess: true,
	}
	done := &types.Stage{ID: 2, Name: "a", Status: enum.CIStatusSuccess}
	require.NoError(t, td.scheduleDownstream(ctx, []*types.Stage{done, waiting}))
	require.Equal(t, enum.CIStatusRunning, waiting.Status, "stage state is refreshed from the database")
	require.Equal(t, "m", waiting.Machine)
	require.Empty(t, h.sched.scheduled, "conflicting stages are not scheduled")

	// same for stages that get skipped
	skipped := &types.Stage{ID: 3, Name: "s", Status: enum.CIStatusWaitingOnDeps, DependsOn: []string{"a"}}
	require.NoError(t, td.cancelDownstream(ctx, []*types.Stage{done, skipped}))
	require.Equal(t, "m", skipped.Machine)

	// a failing resync is only logged
	h.stages.findFn = func(int64) (*types.Stage, error) { return nil, errInfBoom }
	again := &types.Stage{ID: 4, Name: "x", Status: enum.CIStatusWaitingOnDeps, DependsOn: []string{"a"}, OnSuccess: true}
	require.NoError(t, td.scheduleDownstream(ctx, []*types.Stage{done, again}))
	again.Status = enum.CIStatusWaitingOnDeps
	again.OnSuccess = false
	require.NoError(t, td.cancelDownstream(ctx, []*types.Stage{done, again}))
}

func TestInfTeardown_DownstreamErrorsAreAggregated(t *testing.T) {
	ctx := context.Background()
	h := infNewManagerHarness()
	h.stages.updateFn = func(*types.Stage) error { return errInfBoom }
	h.sched.scheduleErr = errInfBoom
	td := &teardown{Stages: h.stages, Scheduler: h.sched}
	done := &types.Stage{ID: 2, Name: "a", Status: enum.CIStatusSuccess}

	waiting := &types.Stage{
		ID: 1, Name: "w", Status: enum.CIStatusWaitingOnDeps, DependsOn: []string{"a"}, OnSuccess: true,
	}
	require.ErrorIs(t, td.scheduleDownstream(ctx, []*types.Stage{done, waiting}), errInfBoom)

	skipped := &types.Stage{ID: 3, Name: "s", Status: enum.CIStatusWaitingOnDeps, DependsOn: []string{"a"}}
	require.ErrorIs(t, td.cancelDownstream(ctx, []*types.Stage{done, skipped}), errInfBoom)
}

func TestInfTeardown_StagesWithPendingDependenciesAreLeftAlone(t *testing.T) {
	ctx := context.Background()
	h := infNewManagerHarness()
	td := &teardown{Stages: h.stages, Scheduler: h.sched}
	running := &types.Stage{ID: 1, Name: "a", Status: enum.CIStatusRunning}
	waiting := &types.Stage{
		ID: 2, Name: "w", Status: enum.CIStatusWaitingOnDeps, DependsOn: []string{"a"}, OnSuccess: true,
	}
	noDeps := &types.Stage{ID: 3, Name: "n", Status: enum.CIStatusWaitingOnDeps}

	require.NoError(t, td.scheduleDownstream(ctx, []*types.Stage{running, waiting, noDeps}))
	require.NoError(t, td.cancelDownstream(ctx, []*types.Stage{running, waiting}))
	require.Equal(t, enum.CIStatusWaitingOnDeps, waiting.Status)
	require.Empty(t, h.stages.updated)
}

func TestInfTeardown_Errors(t *testing.T) {
	ctx := context.Background()
	execution := func() *types.Execution {
		return &types.Execution{ID: 2, RepoID: 3, PipelineID: 5, Status: enum.CIStatusRunning}
	}
	complete := []*types.Stage{{ID: 1, Status: enum.CIStatusSuccess}}

	failures := map[string]func(h *infManagerHarness){
		"execution not found": func(h *infManagerHarness) {
			h.executions.findFn = func(int64) (*types.Execution, error) { return nil, errInfBoom }
		},
		"repo not found": func(h *infManagerHarness) { h.repos.err = errInfBoom },
		"step update":    func(h *infManagerHarness) { h.steps.updateFn = func(*types.Step) error { return errInfBoom } },
		"stage update":   func(h *infManagerHarness) { h.stages.updateFn = func(*types.Stage) error { return errInfBoom } },
		"list stages": func(h *infManagerHarness) {
			h.stages.listWithStepsFn = func(int64) ([]*types.Stage, error) { return nil, errInfBoom }
		},
		"execution update": func(h *infManagerHarness) {
			h.executions.updateFn = func(*types.Execution) error { return errInfBoom }
		},
		"pipeline not found": func(h *infManagerHarness) { h.pipelines.err = errInfBoom },
	}
	for name, mutate := range failures {
		t.Run(name, func(t *testing.T) {
			h := infNewManagerHarness()
			infWireTeardown(h, execution(), complete)
			mutate(h)
			stage := &types.Stage{ID: 1, ExecutionID: 2, Steps: []*types.Step{{ID: 10}}}
			require.ErrorIs(t, h.mgr.AfterStage(ctx, stage), errInfBoom)
		})
	}

	t.Run("version conflict on the execution is not an error", func(t *testing.T) {
		h := infNewManagerHarness()
		infWireTeardown(h, execution(), complete)
		h.executions.updateFn = func(*types.Execution) error { return gitnessstore.ErrVersionConflict }
		require.NoError(t, h.mgr.AfterStage(ctx, &types.Stage{ID: 1, ExecutionID: 2}))
		h.sse.AssertNotCalled(t, "Publish", mock.Anything, enum.SSETypeExecutionCompleted, mock.Anything)
	})
	t.Run("check store failures are tolerated", func(t *testing.T) {
		h := infNewManagerHarness()
		infWireTeardown(h, execution(), complete)
		h.checks.err = errInfBoom
		require.NoError(t, h.mgr.AfterStage(ctx, &types.Stage{ID: 1, ExecutionID: 2}))
	})
	t.Run("failing downstream scheduling aborts the teardown", func(t *testing.T) {
		h := infNewManagerHarness()
		waiting := &types.Stage{ID: 2, Name: "w", Status: enum.CIStatusWaitingOnDeps,
			DependsOn: []string{"a"}, OnSuccess: true}
		stages := []*types.Stage{{ID: 1, Name: "a", Status: enum.CIStatusSuccess}, waiting}
		infWireTeardown(h, execution(), stages)
		h.sched.scheduleErr = errInfBoom
		require.ErrorIs(t, h.mgr.AfterStage(ctx, &types.Stage{ID: 1, ExecutionID: 2}), errInfBoom)
	})
}

func TestInfIsExecutionComplete(t *testing.T) {
	for _, status := range []enum.CIStatus{
		enum.CIStatusPending, enum.CIStatusRunning, enum.CIStatusWaitingOnDeps,
		enum.CIStatusDeclined, enum.CIStatusBlocked,
	} {
		require.False(t, isexecutionComplete([]*types.Stage{{Status: enum.CIStatusSuccess}, {Status: status}}), status)
	}
	require.True(t, isexecutionComplete([]*types.Stage{{Status: enum.CIStatusSuccess}, {Status: enum.CIStatusError}}))
	require.True(t, isexecutionComplete(nil))
}

func TestInfAreDepsComplete(t *testing.T) {
	stages := []*types.Stage{
		{Name: "a", Status: enum.CIStatusSuccess},
		{Name: "b", Status: enum.CIStatusRunning},
		{Name: "c", Status: enum.CIStatusFailure},
	}
	require.True(t, areDepsComplete(&types.Stage{DependsOn: []string{"a", "c"}}, stages))
	require.False(t, areDepsComplete(&types.Stage{DependsOn: []string{"a", "b"}}, stages))
	require.True(t, areDepsComplete(&types.Stage{}, stages))
	require.True(t, areDepsComplete(&types.Stage{DependsOn: []string{"unknown"}}, stages))
}

func TestInfSetup_UpdateExecution(t *testing.T) {
	ctx := context.Background()
	h := infNewManagerHarness()
	s := &setup{Executions: h.executions}

	started, err := s.updateExecution(ctx, &types.Execution{Status: enum.CIStatusRunning})
	require.NoError(t, err)
	require.False(t, started)

	pending := &types.Execution{Status: enum.CIStatusPending}
	started, err = s.updateExecution(ctx, pending)
	require.NoError(t, err)
	require.True(t, started)
	require.Equal(t, enum.CIStatusRunning, pending.Status)

	h.executions.updateFn = func(*types.Execution) error { return gitnessstore.ErrVersionConflict }
	started, err = s.updateExecution(ctx, &types.Execution{Status: enum.CIStatusPending})
	require.NoError(t, err)
	require.False(t, started)

	h.executions.updateFn = func(*types.Execution) error { return errInfBoom }
	_, err = s.updateExecution(ctx, &types.Execution{Status: enum.CIStatusPending})
	require.ErrorIs(t, err, errInfBoom)
}
