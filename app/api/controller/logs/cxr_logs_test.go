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

package logs

import (
	"context"
	"strings"
	"testing"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/app/store/logs"
	"github.com/harness/gitness/livelog"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

const (
	cxrRepoRef  = "root/repo"
	cxrPipeline = "pipe"
)

type cxrEnv struct {
	*cxrBase
	logStore store.LogStore
	stream   livelog.LogStream
	step     *types.Step
}

func cxrNewEnv(ctx context.Context, t *testing.T) *cxrEnv {
	t.Helper()
	b := cxrNewBase(ctx, t)
	repo := b.createRepo(ctx, t, b.createSpace(ctx, t, "root"), "repo")

	pipeline := &types.Pipeline{
		Identifier: cxrPipeline, RepoID: repo.ID, CreatedBy: b.admin.ID, ConfigPath: ".x.yml", DefaultBranch: "main",
	}
	require.NoError(t, database.NewPipelineStore(b.db).Create(ctx, pipeline))
	exec := &types.Execution{
		PipelineID: pipeline.ID, RepoID: repo.ID, CreatedBy: b.admin.ID, Number: 1, Status: enum.CIStatusRunning,
	}
	require.NoError(t, database.NewExecutionStore(b.db).Create(ctx, exec))
	stage := &types.Stage{
		ExecutionID: exec.ID, RepoID: repo.ID, Number: 1, Name: "build", Status: enum.CIStatusRunning,
	}
	stageStore := database.NewStageStore(b.db)
	require.NoError(t, stageStore.Create(ctx, stage))
	// The store does not report the generated ID back on the passed object.
	stage, err := stageStore.FindByNumber(ctx, exec.ID, 1)
	require.NoError(t, err)
	step := &types.Step{StageID: stage.ID, Number: 1, Name: "run", Status: enum.CIStatusRunning}
	stepStore := database.NewStepStore(b.db)
	require.NoError(t, stepStore.Create(ctx, step))
	step, err = stepStore.FindByNumber(ctx, stage.ID, 1)
	require.NoError(t, err)

	return &cxrEnv{
		cxrBase: b, logStore: logs.NewDatabaseLogStore(b.db), stream: livelog.NewMemory(), step: step,
	}
}

func (e *cxrEnv) controller(mode cxrAuthMode) *Controller {
	return NewController(
		cxrAuthorizer{mode: mode}, database.NewExecutionStore(e.db), database.NewPipelineStore(e.db),
		database.NewStageStore(e.db), database.NewStepStore(e.db), e.logStore, e.stream, e.repoFinder)
}

func TestCxrLogsFind(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	c := e.controller(cxrAllow)

	// No logs stored yet.
	_, err := c.Find(ctx, e.session(), cxrRepoRef, cxrPipeline, 1, 1, 1)
	require.ErrorContains(t, err, "could not find logs")

	body := `[{"pos":0,"out":"hello","time":1},{"pos":1,"out":"world","time":2}]`
	require.NoError(t, e.logStore.Create(ctx, e.step.ID, strings.NewReader(body)))
	lines, err := c.Find(ctx, e.session(), cxrRepoRef, cxrPipeline, 1, 1, 1)
	require.NoError(t, err)
	require.Len(t, lines, 2)
	require.Equal(t, "world", lines[1].Message)

	// Invalid JSON content.
	require.NoError(t, e.logStore.Update(ctx, e.step.ID, strings.NewReader("not json")))
	_, err = c.Find(ctx, e.session(), cxrRepoRef, cxrPipeline, 1, 1, 1)
	require.ErrorContains(t, err, "could not unmarshal logs")
}

func TestCxrLogsLookupErrors(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	allow, deny := e.controller(cxrAllow), e.controller(cxrDeny)

	tests := []struct {
		name     string
		ctrl     *Controller
		repoRef  string
		pipeline string
		exec     int64
		stage    int
		step     int
		want     string
	}{
		{"unknown repo", allow, "root/nope", cxrPipeline, 1, 1, 1, "failed to find repo"},
		{"forbidden", deny, cxrRepoRef, cxrPipeline, 1, 1, 1, "failed to authorize"},
		{"unknown pipeline", allow, cxrRepoRef, "nope", 1, 1, 1, "failed to find pipeline"},
		{"unknown execution", allow, cxrRepoRef, cxrPipeline, 9, 1, 1, "failed to find execution"},
		{"unknown stage", allow, cxrRepoRef, cxrPipeline, 1, 9, 1, "failed to find stage"},
		{"unknown step", allow, cxrRepoRef, cxrPipeline, 1, 1, 9, "failed to find step"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.ctrl.Find(ctx, e.session(), tt.repoRef, tt.pipeline, tt.exec, tt.stage, tt.step)
			require.ErrorContains(t, err, tt.want)

			_, _, err = tt.ctrl.Tail(ctx, e.session(), tt.repoRef, tt.pipeline, tt.exec, tt.stage, tt.step)
			require.ErrorContains(t, err, tt.want)
		})
	}
}

func TestCxrLogsTail(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	require.NoError(t, e.stream.Create(ctx, e.step.ID))

	linec, errc, err := e.controller(cxrAllow).Tail(ctx, e.session(), cxrRepoRef, cxrPipeline, 1, 1, 1)
	require.NoError(t, err)
	require.NotNil(t, linec)
	require.NotNil(t, errc)

	require.NoError(t, e.stream.Write(ctx, e.step.ID, &livelog.Line{Number: 0, Message: "hi"}))
	line := <-linec
	require.Equal(t, "hi", line.Message)
}
