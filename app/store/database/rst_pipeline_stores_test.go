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

package database_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/database"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

// rstCIFixture bundles the CI related stores together with a space and a repository to attach pipelines to.
type rstCIFixture struct {
	*rstFixture

	spaceID   int64
	repoID    int64
	pipelines store.PipelineStore
	execs     store.ExecutionStore
	stages    store.StageStore
	steps     store.StepStore
	triggers  store.TriggerStore
}

func rstNewCIFixture(t *testing.T) (context.Context, *rstCIFixture) {
	t.Helper()

	ctx, base := rstNewFixture(t)
	space := base.rstMkSpace(ctx, t, 0, "ci")
	repo := base.rstMkRepo(ctx, t, space.ID, "ci-repo")

	return ctx, &rstCIFixture{
		rstFixture: base,
		spaceID:    space.ID,
		repoID:     repo.ID,
		pipelines:  database.NewPipelineStore(base.db),
		execs:      database.NewExecutionStore(base.db),
		stages:     database.NewStageStore(base.db),
		steps:      database.NewStepStore(base.db),
		triggers:   database.NewTriggerStore(base.db),
	}
}

func (f *rstCIFixture) rstMkPipeline(ctx context.Context, t *testing.T, ident string) *types.Pipeline {
	t.Helper()

	p := &types.Pipeline{
		Description: "desc " + ident, Identifier: ident, CreatedBy: userID, RepoID: f.repoID,
		DefaultBranch: "main", ConfigPath: ".harness/" + ident + ".yaml", Created: 1, Updated: 1,
	}
	require.NoError(t, f.pipelines.Create(ctx, p))
	require.NotZero(t, p.ID)

	return p
}

func (f *rstCIFixture) rstMkExecution(
	ctx context.Context, t *testing.T, p *types.Pipeline, number int64, started int64,
) *types.Execution {
	t.Helper()

	e := &types.Execution{
		PipelineID: p.ID, RepoID: f.repoID, CreatedBy: userID, Number: number, Status: enum.CIStatusPending,
		Event: enum.TriggerEventPush, Action: enum.TriggerActionBranchCreated, Trigger: "@hook",
		Title: fmt.Sprintf("execution %d", number), Params: map[string]string{"k": "v"},
		Started: started, Finished: started + 10, Created: started, Updated: started,
	}
	require.NoError(t, f.execs.Create(ctx, e))
	require.NotZero(t, e.ID)

	return e
}

func TestRstPipelineStore(t *testing.T) {
	ctx, f := rstNewCIFixture(t)

	build := f.rstMkPipeline(ctx, t, "build")
	deploy := f.rstMkPipeline(ctx, t, "deploy")

	t.Run("duplicate identifier in repo", func(t *testing.T) {
		err := f.pipelines.Create(ctx, &types.Pipeline{Identifier: "build", RepoID: f.repoID, CreatedBy: userID})
		require.ErrorIs(t, err, gitness_store.ErrDuplicate)
	})

	t.Run("find", func(t *testing.T) {
		got, err := f.pipelines.Find(ctx, build.ID)
		require.NoError(t, err)
		require.Equal(t, "build", got.Identifier)
		require.Equal(t, ".harness/build.yaml", got.ConfigPath)

		got, err = f.pipelines.FindByIdentifier(ctx, f.repoID, "deploy")
		require.NoError(t, err)
		require.Equal(t, deploy.ID, got.ID)

		_, err = f.pipelines.Find(ctx, 9999)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
		_, err = f.pipelines.FindByIdentifier(ctx, f.repoID, "missing")
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("update detects stale version", func(t *testing.T) {
		stale := *build

		build.Description = rstValChanged
		require.NoError(t, f.pipelines.Update(ctx, build))
		require.Equal(t, int64(1), build.Version)

		got, err := f.pipelines.Find(ctx, build.ID)
		require.NoError(t, err)
		require.Equal(t, rstValChanged, got.Description)

		stale.Description = rstValStale
		require.ErrorIs(t, f.pipelines.Update(ctx, &stale), gitness_store.ErrVersionConflict)
	})

	t.Run("update opt lock retries on conflict", func(t *testing.T) {
		current, err := f.pipelines.Find(ctx, deploy.ID)
		require.NoError(t, err)

		concurrent := *current
		concurrent.Disabled = true
		require.NoError(t, f.pipelines.Update(ctx, &concurrent))

		calls := 0
		updated, err := f.pipelines.UpdateOptLock(ctx, current, func(p *types.Pipeline) error {
			calls++
			p.Description = rstValOptLock
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, 2, calls)
		require.True(t, updated.Disabled)
		require.Equal(t, rstValOptLock, updated.Description)

		boom := errors.New("boom")
		_, err = f.pipelines.UpdateOptLock(ctx, current, func(*types.Pipeline) error { return boom })
		require.ErrorIs(t, err, boom)
	})

	t.Run("increment seq num retries on conflict", func(t *testing.T) {
		current, err := f.pipelines.Find(ctx, build.ID)
		require.NoError(t, err)
		stale := *current

		// bump the stored version so the first attempt on the stale copy conflicts.
		require.NoError(t, f.pipelines.Update(ctx, current))

		got, err := f.pipelines.IncrementSeqNum(ctx, &stale)
		require.NoError(t, err)
		require.Equal(t, int64(1), got.Seq)

		got, err = f.pipelines.IncrementSeqNum(ctx, got)
		require.NoError(t, err)
		require.Equal(t, int64(2), got.Seq)

		stored, err := f.pipelines.Find(ctx, build.ID)
		require.NoError(t, err)
		require.Equal(t, int64(2), stored.Seq)
	})

	t.Run("list and count", func(t *testing.T) {
		got, err := f.pipelines.List(ctx, f.repoID, &types.ListPipelinesFilter{})
		require.NoError(t, err)
		require.Len(t, got, 2)

		got, err = f.pipelines.List(ctx, f.repoID, &types.ListPipelinesFilter{
			ListQueryFilter: types.ListQueryFilter{Query: "DEP"},
		})
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, "deploy", got[0].Identifier)

		got, err = f.pipelines.List(ctx, f.repoID, &types.ListPipelinesFilter{
			ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 2, Size: 1}},
		})
		require.NoError(t, err)
		require.Len(t, got, 1)

		count, err := f.pipelines.Count(ctx, f.repoID, &types.ListPipelinesFilter{})
		require.NoError(t, err)
		require.Equal(t, int64(2), count)

		// repo id 0 counts every pipeline in the system.
		count, err = f.pipelines.Count(ctx, 0, &types.ListPipelinesFilter{
			ListQueryFilter: types.ListQueryFilter{Query: "build"},
		})
		require.NoError(t, err)
		require.Equal(t, int64(1), count)
	})

	t.Run("list and count in space", func(t *testing.T) {
		got, err := f.pipelines.ListInSpace(ctx, f.spaceID, types.ListPipelinesFilter{})
		require.NoError(t, err)
		require.Len(t, got, 2)
		for _, p := range got {
			require.Equal(t, "ci-repo", p.RepoUID)
		}

		got, err = f.pipelines.ListInSpace(ctx, f.spaceID, types.ListPipelinesFilter{
			ListQueryFilter: types.ListQueryFilter{Query: "build"},
		})
		require.NoError(t, err)
		require.Len(t, got, 1)

		count, err := f.pipelines.CountInSpace(ctx, f.spaceID, types.ListPipelinesFilter{})
		require.NoError(t, err)
		require.Equal(t, int64(2), count)

		count, err = f.pipelines.CountInSpace(ctx, f.spaceID, types.ListPipelinesFilter{
			ListQueryFilter: types.ListQueryFilter{Query: "nothing"},
		})
		require.NoError(t, err)
		require.Zero(t, count)

		other, err := f.pipelines.ListInSpace(ctx, 9999, types.ListPipelinesFilter{})
		require.NoError(t, err)
		require.Empty(t, other)
	})

	t.Run("delete", func(t *testing.T) {
		require.NoError(t, f.pipelines.Delete(ctx, deploy.ID))
		_, err := f.pipelines.Find(ctx, deploy.ID)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

		require.NoError(t, f.pipelines.DeleteByIdentifier(ctx, f.repoID, "build"))
		_, err = f.pipelines.FindByIdentifier(ctx, f.repoID, "build")
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})
}

func TestRstPipelineStore_ListLatest(t *testing.T) {
	ctx, f := rstNewCIFixture(t)

	build := f.rstMkPipeline(ctx, t, "build")
	f.rstMkPipeline(ctx, t, "idle")

	f.rstMkExecution(ctx, t, build, 1, 100)
	latest := f.rstMkExecution(ctx, t, build, 2, 200)

	got, err := f.pipelines.ListLatest(ctx, f.repoID, &types.ListPipelinesFilter{})
	require.NoError(t, err)
	require.Len(t, got, 2)

	byIdent := map[string]*types.Pipeline{}
	for _, p := range got {
		byIdent[p.Identifier] = p
	}

	require.Nil(t, byIdent["idle"].Execution)

	require.NotNil(t, byIdent["build"].Execution)
	require.Equal(t, latest.ID, byIdent["build"].Execution.ID)
	require.Equal(t, int64(2), byIdent["build"].Execution.Number)

	got, err = f.pipelines.ListLatest(ctx, f.repoID, &types.ListPipelinesFilter{
		ListQueryFilter: types.ListQueryFilter{Query: "idle"},
	})
	require.NoError(t, err)
	require.Len(t, got, 1)
}

func TestRstExecutionStore(t *testing.T) {
	ctx, f := rstNewCIFixture(t)

	build := f.rstMkPipeline(ctx, t, "build")
	deploy := f.rstMkPipeline(ctx, t, "deploy")

	first := f.rstMkExecution(ctx, t, build, 1, 100)
	second := f.rstMkExecution(ctx, t, build, 2, 300)
	third := f.rstMkExecution(ctx, t, build, 3, 200)
	other := f.rstMkExecution(ctx, t, deploy, 1, 400)

	t.Run("duplicate number in pipeline", func(t *testing.T) {
		err := f.execs.Create(ctx, &types.Execution{PipelineID: build.ID, RepoID: f.repoID, Number: 1})
		require.ErrorIs(t, err, gitness_store.ErrDuplicate)
	})

	t.Run("find", func(t *testing.T) {
		got, err := f.execs.Find(ctx, first.ID)
		require.NoError(t, err)
		require.Equal(t, int64(1), got.Number)
		require.Equal(t, map[string]string{"k": "v"}, got.Params)
		require.Equal(t, enum.TriggerEventPush, got.Event)
		require.Equal(t, enum.TriggerActionBranchCreated, got.Action)

		got, err = f.execs.FindByNumber(ctx, build.ID, 2)
		require.NoError(t, err)
		require.Equal(t, second.ID, got.ID)

		_, err = f.execs.Find(ctx, 9999)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
		_, err = f.execs.FindByNumber(ctx, build.ID, 99)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("update", func(t *testing.T) {
		stale := *first

		first.Status = enum.CIStatusRunning
		first.Error = "oops"
		first.Finished = 999
		first.Stages = []*types.Stage{{Name: "kept"}}
		require.NoError(t, f.execs.Update(ctx, first))
		require.Equal(t, int64(1), first.Version)
		require.Len(t, first.Stages, 1, "stages are not persisted but must survive the update")

		got, err := f.execs.Find(ctx, first.ID)
		require.NoError(t, err)
		require.Equal(t, enum.CIStatusRunning, got.Status)
		require.Equal(t, "oops", got.Error)
		require.Equal(t, int64(999), got.Finished)

		stale.Status = enum.CIStatusFailure
		require.ErrorIs(t, f.execs.Update(ctx, &stale), gitness_store.ErrVersionConflict)
	})

	t.Run("list by pipeline is newest number first", func(t *testing.T) {
		got, err := f.execs.List(ctx, build.ID, types.Pagination{})
		require.NoError(t, err)
		require.Equal(t, []int64{3, 2, 1}, []int64{got[0].Number, got[1].Number, got[2].Number})

		got, err = f.execs.List(ctx, build.ID, types.Pagination{Page: 2, Size: 2})
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, int64(1), got[0].Number)
	})

	t.Run("list in space", func(t *testing.T) {
		got, err := f.execs.ListInSpace(ctx, f.spaceID, types.ListExecutionsFilter{
			Sort: enum.ExecutionSortStarted, Order: enum.OrderDesc,
		})
		require.NoError(t, err)
		require.Equal(t, []int64{other.ID, second.ID, third.ID, first.ID},
			[]int64{got[0].ID, got[1].ID, got[2].ID, got[3].ID})
		require.Equal(t, "deploy", got[0].PipelineUID)
		require.Equal(t, "ci-repo", got[0].RepoUID)

		got, err = f.execs.ListInSpace(ctx, f.spaceID, types.ListExecutionsFilter{
			Sort: enum.ExecutionSortFinished, Order: enum.OrderAsc, PipelineIdentifier: "build",
		})
		require.NoError(t, err)
		require.Len(t, got, 3)

		got, err = f.execs.ListInSpace(ctx, f.spaceID, types.ListExecutionsFilter{
			Sort: enum.ExecutionSortStarted, ListQueryFilter: types.ListQueryFilter{Query: "DEPL"},
		})
		require.NoError(t, err)
		require.Len(t, got, 1)

		count, err := f.execs.CountInSpace(ctx, f.spaceID, types.ListExecutionsFilter{})
		require.NoError(t, err)
		require.Equal(t, int64(4), count)

		count, err = f.execs.CountInSpace(ctx, f.spaceID, types.ListExecutionsFilter{
			PipelineIdentifier: "deploy", ListQueryFilter: types.ListQueryFilter{Query: "dep"},
		})
		require.NoError(t, err)
		require.Equal(t, int64(1), count)
	})

	t.Run("list by pipeline ids", func(t *testing.T) {
		got, err := f.execs.ListByPipelineIDs(ctx, []int64{build.ID, deploy.ID}, 2)
		require.NoError(t, err)
		require.Len(t, got[build.ID], 2)
		require.Len(t, got[deploy.ID], 1)

		numbers := []int64{got[build.ID][0].Number, got[build.ID][1].Number}
		require.ElementsMatch(t, []int64{3, 2}, numbers)

		empty, err := f.execs.ListByPipelineIDs(ctx, nil, 2)
		require.NoError(t, err)
		require.Empty(t, empty)
	})

	t.Run("count and delete", func(t *testing.T) {
		count, err := f.execs.Count(ctx, build.ID)
		require.NoError(t, err)
		require.Equal(t, int64(3), count)

		count, err = f.execs.Count(ctx, 0)
		require.NoError(t, err)
		require.Equal(t, int64(4), count)

		require.NoError(t, f.execs.Delete(ctx, build.ID, 3))
		_, err = f.execs.FindByNumber(ctx, build.ID, 3)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})
}

func TestRstStageAndStepStores(t *testing.T) {
	ctx, f := rstNewCIFixture(t)

	p := f.rstMkPipeline(ctx, t, "build")
	exec := f.rstMkExecution(ctx, t, p, 1, 100)
	otherExec := f.rstMkExecution(ctx, t, p, 2, 200)

	mkStage := func(execID int64, number int64, status enum.CIStatus) {
		st := &types.Stage{
			ExecutionID: execID, RepoID: f.repoID, Number: number, Name: fmt.Sprintf("stage-%d", number),
			Kind: "pipeline", Type: "ci", Status: status, OS: "linux", Arch: "amd64", Limit: 1,
			OnSuccess: true, DependsOn: []string{"prev"}, Labels: map[string]string{"a": "b"},
			Created: 1, Updated: 1,
		}
		require.NoError(t, f.stages.Create(ctx, st))
	}

	mkStage(exec.ID, 1, enum.CIStatusSuccess)
	mkStage(exec.ID, 2, enum.CIStatusRunning)
	mkStage(otherExec.ID, 1, enum.CIStatusPending)

	t.Run("duplicate stage number", func(t *testing.T) {
		err := f.stages.Create(ctx, &types.Stage{ExecutionID: exec.ID, RepoID: f.repoID, Number: 1})
		require.ErrorIs(t, err, gitness_store.ErrDuplicate)
	})

	running, err := f.stages.FindByNumber(ctx, exec.ID, 2)
	require.NoError(t, err)
	require.Equal(t, "stage-2", running.Name)
	require.Equal(t, []string{"prev"}, running.DependsOn)
	require.Equal(t, map[string]string{"a": "b"}, running.Labels)
	require.True(t, running.OnSuccess)

	t.Run("find", func(t *testing.T) {
		got, err := f.stages.Find(ctx, running.ID)
		require.NoError(t, err)
		require.Equal(t, running.ID, got.ID)

		_, err = f.stages.Find(ctx, 9999)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
		_, err = f.stages.FindByNumber(ctx, exec.ID, 99)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("list", func(t *testing.T) {
		got, err := f.stages.List(ctx, exec.ID)
		require.NoError(t, err)
		require.Len(t, got, 2)
		require.Equal(t, int64(1), got[0].Number)
		require.Equal(t, int64(2), got[1].Number)

		incomplete, err := f.stages.ListIncomplete(ctx)
		require.NoError(t, err)
		require.Len(t, incomplete, 2)
		for _, st := range incomplete {
			require.Contains(t, []enum.CIStatus{enum.CIStatusPending, enum.CIStatusRunning}, st.Status)
		}
	})

	t.Run("update", func(t *testing.T) {
		stale := *running

		running.Status = enum.CIStatusSuccess
		running.Machine = "node-1"
		running.ExitCode = 3
		running.Error = "late failure"
		running.DependsOn = []string{"a", "b"}
		running.Labels = map[string]string{"x": "y"}
		running.Steps = []*types.Step{{Name: "kept"}}
		require.NoError(t, f.stages.Update(ctx, running))
		require.Equal(t, int64(1), running.Version)
		require.Len(t, running.Steps, 1)

		got, err := f.stages.Find(ctx, running.ID)
		require.NoError(t, err)
		require.Equal(t, enum.CIStatusSuccess, got.Status)
		require.Equal(t, "node-1", got.Machine)
		require.Equal(t, 3, got.ExitCode)
		require.Equal(t, []string{"a", "b"}, got.DependsOn)
		require.Equal(t, map[string]string{"x": "y"}, got.Labels)

		stale.Status = enum.CIStatusFailure
		require.ErrorIs(t, f.stages.Update(ctx, &stale), gitness_store.ErrVersionConflict)
	})

	t.Run("steps", func(t *testing.T) {
		mkStep := func(stageID, number int64, status enum.CIStatus) *types.Step {
			s := &types.Step{
				StageID: stageID, Number: number, Name: fmt.Sprintf("step-%d", number), Status: status,
				Image: "alpine", Detached: number == 2, Schema: "schema", DependsOn: []string{"dep"},
			}
			require.NoError(t, f.steps.Create(ctx, s))
			require.NotZero(t, s.ID)

			return s
		}

		first := mkStep(running.ID, 1, enum.CIStatusSuccess)
		mkStep(running.ID, 2, enum.CIStatusPending)

		err := f.steps.Create(ctx, &types.Step{StageID: running.ID, Number: 1, Name: "dup"})
		require.ErrorIs(t, err, gitness_store.ErrDuplicate)

		got, err := f.steps.FindByNumber(ctx, running.ID, 2)
		require.NoError(t, err)
		require.Equal(t, "step-2", got.Name)
		require.True(t, got.Detached)
		require.Equal(t, []string{"dep"}, got.DependsOn)

		_, err = f.steps.FindByNumber(ctx, running.ID, 99)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

		stale := *first
		first.Status = enum.CIStatusFailure
		first.ExitCode = 2
		first.Error = "failed"
		first.Image = "busybox"
		first.DependsOn = []string{"x"}
		require.NoError(t, f.steps.Update(ctx, first))
		require.Equal(t, int64(1), first.Version)

		got, err = f.steps.FindByNumber(ctx, running.ID, 1)
		require.NoError(t, err)
		require.Equal(t, enum.CIStatusFailure, got.Status)
		require.Equal(t, 2, got.ExitCode)
		require.Equal(t, "busybox", got.Image)
		require.Equal(t, []string{"x"}, got.DependsOn)

		stale.Name = rstValStale
		require.ErrorIs(t, f.steps.Update(ctx, &stale), gitness_store.ErrVersionConflict)
	})

	t.Run("list with steps", func(t *testing.T) {
		got, err := f.stages.ListWithSteps(ctx, exec.ID)
		require.NoError(t, err)
		require.Len(t, got, 2)

		// the first stage has no steps, the second one has the two created above.
		require.Empty(t, got[0].Steps)
		require.Len(t, got[1].Steps, 2)
		require.Equal(t, "step-1", got[1].Steps[0].Name)
		require.Equal(t, []string{"x"}, got[1].Steps[0].DependsOn)
		require.Equal(t, []string{"a", "b"}, got[1].DependsOn)

		none, err := f.stages.ListWithSteps(ctx, 9999)
		require.NoError(t, err)
		require.Empty(t, none)
	})
}

func TestRstTriggerStore(t *testing.T) {
	ctx, f := rstNewCIFixture(t)
	p := f.rstMkPipeline(ctx, t, "build")

	mk := func(ident string, disabled bool) *types.Trigger {
		tr := &types.Trigger{
			Identifier: ident, Description: "desc " + ident, Type: "webhook", Secret: "secret", PipelineID: p.ID,
			RepoID: f.repoID, CreatedBy: userID, Disabled: disabled, Created: 1, Updated: 1,
			Actions: []enum.TriggerAction{enum.TriggerActionBranchCreated, enum.TriggerActionTagCreated},
		}
		require.NoError(t, f.triggers.Create(ctx, tr))

		return tr
	}

	on := mk("on-push", false)
	mk("nightly", true)

	// Create does not report the generated id back, so reload to get an updatable trigger.
	off, err := f.triggers.FindByIdentifier(ctx, p.ID, "nightly")
	require.NoError(t, err)
	require.NotZero(t, off.ID)

	t.Run("duplicate identifier", func(t *testing.T) {
		err := f.triggers.Create(ctx, &types.Trigger{
			Identifier: "on-push", PipelineID: p.ID, RepoID: f.repoID, CreatedBy: userID,
		})
		require.ErrorIs(t, err, gitness_store.ErrDuplicate)
	})

	t.Run("find by identifier", func(t *testing.T) {
		got, err := f.triggers.FindByIdentifier(ctx, p.ID, "on-push")
		require.NoError(t, err)
		require.Equal(t, "desc on-push", got.Description)
		require.Equal(t, []enum.TriggerAction{enum.TriggerActionBranchCreated, enum.TriggerActionTagCreated}, got.Actions)
		require.False(t, got.Disabled)

		_, err = f.triggers.FindByIdentifier(ctx, p.ID, "missing")
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("update and opt lock", func(t *testing.T) {
		stale := *off

		off.Description = "new description"
		off.Disabled = false
		off.Actions = []enum.TriggerAction{enum.TriggerActionPullReqCreated}
		require.NoError(t, f.triggers.Update(ctx, off))
		require.Equal(t, int64(1), off.Version)

		got, err := f.triggers.FindByIdentifier(ctx, p.ID, "nightly")
		require.NoError(t, err)
		require.Equal(t, "new description", got.Description)
		require.False(t, got.Disabled)
		require.Equal(t, []enum.TriggerAction{enum.TriggerActionPullReqCreated}, got.Actions)

		stale.Description = rstValStale
		require.ErrorIs(t, f.triggers.Update(ctx, &stale), gitness_store.ErrVersionConflict)

		calls := 0
		updated, err := f.triggers.UpdateOptLock(ctx, &stale, func(tr *types.Trigger) error {
			calls++
			tr.Disabled = true
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, 2, calls)
		require.True(t, updated.Disabled)
		// the retry starts from the stored state, which includes the earlier update.
		require.Equal(t, "new description", updated.Description)

		boom := errors.New("boom")
		_, err = f.triggers.UpdateOptLock(ctx, updated, func(*types.Trigger) error { return boom })
		require.ErrorIs(t, err, boom)
	})

	t.Run("list and count", func(t *testing.T) {
		got, err := f.triggers.List(ctx, p.ID, types.ListQueryFilter{})
		require.NoError(t, err)
		require.Len(t, got, 2)

		got, err = f.triggers.List(ctx, p.ID, types.ListQueryFilter{Query: "NIGHT"})
		require.NoError(t, err)
		require.Len(t, got, 1)

		got, err = f.triggers.List(ctx, p.ID, types.ListQueryFilter{Pagination: types.Pagination{Page: 2, Size: 1}})
		require.NoError(t, err)
		require.Len(t, got, 1)

		count, err := f.triggers.Count(ctx, p.ID, types.ListQueryFilter{})
		require.NoError(t, err)
		require.Equal(t, int64(2), count)

		count, err = f.triggers.Count(ctx, p.ID, types.ListQueryFilter{Query: "push"})
		require.NoError(t, err)
		require.Equal(t, int64(1), count)
	})

	t.Run("list all enabled", func(t *testing.T) {
		// "nightly" was disabled again by the opt lock update above.
		got, err := f.triggers.ListAllEnabled(ctx, f.repoID)
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, "on-push", got[0].Identifier)

		none, err := f.triggers.ListAllEnabled(ctx, 9999)
		require.NoError(t, err)
		require.Empty(t, none)
	})

	t.Run("delete", func(t *testing.T) {
		require.NoError(t, f.triggers.DeleteByIdentifier(ctx, p.ID, on.Identifier))
		_, err := f.triggers.FindByIdentifier(ctx, p.ID, on.Identifier)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})
}
