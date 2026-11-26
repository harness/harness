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

package triggerer

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/pipeline/converter"
	"github.com/harness/gitness/app/pipeline/file"
	"github.com/harness/gitness/app/pipeline/scheduler"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

type infRepoStore struct {
	store.RepoStore

	repo *types.Repository
	err  error
}

func (s *infRepoStore) Find(context.Context, int64) (*types.Repository, error) {
	return s.repo, s.err
}

type infPublicAccess struct {
	publicaccess.Service

	err error
}

func (s *infPublicAccess) Get(context.Context, enum.PublicResourceType, string) (bool, error) {
	return false, s.err
}

type infFileService struct {
	data string
	err  error
}

func (s *infFileService) Get(context.Context, *types.Repository, string, string) (*file.File, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &file.File{Data: []byte(s.data)}, nil
}

type infConverter struct {
	err error
}

func (c *infConverter) Convert(_ context.Context, args *converter.ConvertArgs) (*file.File, error) {
	if c.err != nil {
		return nil, c.err
	}
	return args.File, nil
}

type infPipelineStore struct {
	store.PipelineStore

	incrementErr error
}

func (s *infPipelineStore) IncrementSeqNum(_ context.Context, p *types.Pipeline) (*types.Pipeline, error) {
	if s.incrementErr != nil {
		return nil, s.incrementErr
	}
	cp := *p
	cp.Seq++
	return &cp, nil
}

type infExecutionStore struct {
	store.ExecutionStore

	created []*types.Execution
	err     error
}

func (s *infExecutionStore) Create(_ context.Context, e *types.Execution) error {
	if s.err != nil {
		return s.err
	}
	e.ID = int64(len(s.created) + 100)
	s.created = append(s.created, e)
	return nil
}

type infStageStore struct {
	store.StageStore

	created []*types.Stage
	err     error
}

func (s *infStageStore) Create(_ context.Context, st *types.Stage) error {
	if s.err != nil {
		return s.err
	}
	s.created = append(s.created, st)
	return nil
}

type infCheckStore struct {
	store.CheckStore

	upserted []*types.Check
	err      error
}

func (s *infCheckStore) Upsert(_ context.Context, c *types.Check) error {
	if s.err != nil {
		return s.err
	}
	s.upserted = append(s.upserted, c)
	return nil
}

type infTx struct{}

func (infTx) WithTx(ctx context.Context, fn func(context.Context) error, _ ...any) error {
	return fn(ctx)
}

type infScheduler struct {
	scheduler.Scheduler

	scheduled []*types.Stage
	err       error
}

func (s *infScheduler) Schedule(_ context.Context, st *types.Stage) error {
	if s.err != nil {
		return s.err
	}
	s.scheduled = append(s.scheduled, st)
	return nil
}

type infURLProvider struct {
	url.Provider
}

func (infURLProvider) GenerateUIBuildURL(_ context.Context, repoPath, pipeline string, seq int64) string {
	return "https://ui.invalid/" + repoPath + "/" + pipeline + "/" + string(rune('0'+seq))
}

type infHarness struct {
	repos      *infRepoStore
	access     *infPublicAccess
	files      *infFileService
	conv       *infConverter
	pipelines  *infPipelineStore
	executions *infExecutionStore
	stages     *infStageStore
	checks     *infCheckStore
	sched      *infScheduler
	trig       Triggerer
}

func infNewHarness(fileContent string) *infHarness {
	h := &infHarness{
		repos:      &infRepoStore{repo: &types.Repository{ID: 3, Path: "space/repo", Identifier: "repo", ParentID: 2}},
		access:     &infPublicAccess{},
		files:      &infFileService{data: fileContent},
		conv:       &infConverter{},
		pipelines:  &infPipelineStore{},
		executions: &infExecutionStore{},
		stages:     &infStageStore{},
		checks:     &infCheckStore{},
		sched:      &infScheduler{},
	}
	h.trig = New(h.executions, h.checks, h.stages, h.pipelines, infTx{}, h.repos, infURLProvider{}, h.sched,
		h.files, h.conv, nil, nil, h.access)
	return h
}

func infPipeline() *types.Pipeline {
	return &types.Pipeline{ID: 5, RepoID: 3, Identifier: "build", ConfigPath: ".harness/ci.yaml", Seq: 1}
}

func infHook() *Hook {
	return &Hook{
		Trigger:     "@hook",
		TriggeredBy: 9,
		Action:      enum.TriggerActionBranchUpdated,
		Ref:         "refs/heads/main",
		Target:      "main",
		After:       "abc123",
		Title:       "a title",
	}
}

const infDroneSingle = `kind: pipeline
type: docker
name: build
steps:
- name: test
  image: alpine
  commands:
  - echo hello
`

const infDroneChain = `kind: pipeline
type: docker
name: first
steps:
- name: one
  image: alpine
  commands: [echo 1]
---
kind: pipeline
type: docker
name: second
depends_on: [first]
steps:
- name: two
  image: alpine
  commands: [echo 2]
`

func TestInfTrigger_DroneYamlCreatesExecutionAndStages(t *testing.T) {
	h := infNewHarness(infDroneSingle)
	exec, err := h.trig.Trigger(context.Background(), infPipeline(), infHook())
	require.NoError(t, err)
	require.NotNil(t, exec)

	require.Equal(t, enum.CIStatusPending, exec.Status)
	require.EqualValues(t, 2, exec.Number, "sequence number is taken from the incremented pipeline")
	require.Equal(t, enum.TriggerEventPush, exec.Event)
	require.EqualValues(t, 9, exec.CreatedBy)
	require.EqualValues(t, 100, exec.ID)
	require.Contains(t, exec.Params["DRONE_BUILD_LINK"], "space/repo/build/")

	require.Len(t, h.stages.created, 1)
	stage := h.stages.created[0]
	require.Equal(t, "build", stage.Name)
	require.Equal(t, enum.CIStatusPending, stage.Status)
	require.Equal(t, "docker", stage.Type)
	require.Equal(t, "linux", stage.OS)
	require.Equal(t, "amd64", stage.Arch)
	require.EqualValues(t, 100, stage.ExecutionID)
	require.True(t, stage.OnSuccess)
	require.False(t, stage.OnFailure)

	require.Len(t, h.sched.scheduled, 1)
	require.Len(t, h.checks.upserted, 1)
}

func TestInfTrigger_DependentStagesWaitForDependencies(t *testing.T) {
	h := infNewHarness(infDroneChain)
	exec, err := h.trig.Trigger(context.Background(), infPipeline(), infHook())
	require.NoError(t, err)
	require.NotNil(t, exec)

	require.Len(t, h.stages.created, 2)
	require.Equal(t, enum.CIStatusPending, h.stages.created[0].Status)
	require.Equal(t, enum.CIStatusWaitingOnDeps, h.stages.created[1].Status)
	require.Equal(t, []string{"first"}, h.stages.created[1].DependsOn)
	require.Len(t, h.sched.scheduled, 1, "only stages without pending dependencies are scheduled")
	require.Equal(t, "first", h.sched.scheduled[0].Name)
}

func TestInfTrigger_SkippedDependenciesDoNotBlockStages(t *testing.T) {
	src := `kind: pipeline
type: docker
name: first
trigger:
  branch: [release]
steps:
- name: one
  image: alpine
  commands: [echo 1]
---
kind: pipeline
type: docker
name: second
depends_on: [first]
steps:
- name: two
  image: alpine
  commands: [echo 2]
`
	h := infNewHarness(src)
	exec, err := h.trig.Trigger(context.Background(), infPipeline(), infHook())
	require.NoError(t, err)
	require.NotNil(t, exec)
	require.Len(t, h.stages.created, 1)
	require.Equal(t, "second", h.stages.created[0].Name)
	require.Equal(t, enum.CIStatusPending, h.stages.created[0].Status)
	require.Empty(t, h.stages.created[0].DependsOn)
}

func TestInfTrigger_PipelinesNotMatchingTheTriggerAreSkipped(t *testing.T) {
	tests := []struct {
		name    string
		trigger string
	}{
		{"branch", "branch: [release]"},
		{"event", "event: [tag]"},
		{"action", "action: [pullreq_created]"},
		{"ref", "ref: [refs/tags/*]"},
		{"repo", "repo: [other/repo]"},
		{"cron", "cron: [nightly]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := infDroneSingle + "trigger:\n  " + tt.trigger + "\n"
			h := infNewHarness(src)
			exec, err := h.trig.Trigger(context.Background(), infPipeline(), infHook())
			require.NoError(t, err)
			require.Nil(t, exec)
			require.Empty(t, h.executions.created)
			require.Empty(t, h.sched.scheduled)
		})
	}
}

func TestInfTrigger_MatchingTriggersCreateAnExecution(t *testing.T) {
	src := infDroneSingle + `trigger:
  branch: [main]
  event: [push]
  action: [branch_updated]
  ref: [refs/heads/*]
  repo: [space/repo]
`
	h := infNewHarness(src)
	exec, err := h.trig.Trigger(context.Background(), infPipeline(), infHook())
	require.NoError(t, err)
	require.NotNil(t, exec)
}

func TestInfTrigger_StatusTriggerDecidesOnFailureFlag(t *testing.T) {
	src := infDroneSingle + `trigger:
  status: [failure]
`
	h := infNewHarness(src)
	_, err := h.trig.Trigger(context.Background(), infPipeline(), infHook())
	require.NoError(t, err)
	require.Len(t, h.stages.created, 1)
	require.False(t, h.stages.created[0].OnSuccess)
	require.True(t, h.stages.created[0].OnFailure)
}

func TestInfTrigger_InvalidDefinitionsCreateErroredExecutions(t *testing.T) {
	cycle := `kind: pipeline
type: docker
name: a
depends_on: [b]
steps:
- name: s
  image: alpine
  commands: [echo]
---
kind: pipeline
type: docker
name: b
depends_on: [a]
steps:
- name: s
  image: alpine
  commands: [echo]
`
	tests := []struct {
		name    string
		file    string
		convErr error
	}{
		{name: "conversion failure", file: infDroneSingle, convErr: errors.New("conversion failed")},
		{name: "unparsable yaml", file: "kind: pipeline\n  bad: [indent"},
		{
			name: "lint failure",
			file: infLintFailure,
		},
		{name: "unknown or cyclical dependency", file: cycle},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := infNewHarness(tt.file)
			h.conv.err = tt.convErr
			exec, err := h.trig.Trigger(context.Background(), infPipeline(), infHook())
			require.NoError(t, err)
			require.NotNil(t, exec)
			require.Equal(t, enum.CIStatusError, exec.Status)
			require.NotEmpty(t, exec.Error)
			require.NotZero(t, exec.Finished)
			require.Empty(t, h.stages.created)
			require.Empty(t, h.sched.scheduled)
			require.Len(t, h.checks.upserted, 1, "errored executions are reported as checks")
		})
	}
}

func TestInfTrigger_Errors(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(h *infHarness)
		wantErr string
	}{
		{"repo lookup", func(h *infHarness) { h.repos.err = errors.New("no repo") }, "no repo"},
		{"public access", func(h *infHarness) { h.access.err = errors.New("pa") }, "could not check if repo is public"},
		{"file lookup", func(h *infHarness) { h.files.err = errors.New("no file") }, "no file"},
		{"sequence number", func(h *infHarness) { h.pipelines.incrementErr = errors.New("conflict") }, "conflict"},
		{"execution create", func(h *infHarness) { h.executions.err = errors.New("db") }, "db"},
		{"stage create", func(h *infHarness) { h.stages.err = errors.New("stage db") }, "stage db"},
		{"schedule", func(h *infHarness) { h.sched.err = errors.New("queue full") }, "queue full"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := infNewHarness(infDroneSingle)
			tt.setup(h)
			exec, err := h.trig.Trigger(context.Background(), infPipeline(), infHook())
			require.ErrorContains(t, err, tt.wantErr)
			require.Nil(t, exec)
		})
	}
}

func TestInfTrigger_CheckStoreFailureIsTolerated(t *testing.T) {
	h := infNewHarness(infDroneSingle)
	h.checks.err = errors.New("check db down")
	exec, err := h.trig.Trigger(context.Background(), infPipeline(), infHook())
	require.NoError(t, err)
	require.NotNil(t, exec)
	require.Len(t, h.sched.scheduled, 1)
}

const infLintFailure = "kind: pipeline\ntype: docker\nname: build\nsteps:\n- name: s\n  image: alpine\n" +
	"---\n" + "kind: pipeline\ntype: docker\nname: build\nsteps:\n- name: s\n  image: alpine\n"

func TestInfTrigger_ErroredExecutionStoreFailures(t *testing.T) {
	h := infNewHarness(infLintFailure)
	h.pipelines.incrementErr = errors.New("conflict")
	_, err := h.trig.Trigger(context.Background(), infPipeline(), infHook())
	require.ErrorContains(t, err, "conflict")

	h = infNewHarness(infLintFailure)
	h.executions.err = errors.New("db")
	_, err = h.trig.Trigger(context.Background(), infPipeline(), infHook())
	require.ErrorContains(t, err, "db")
}

func TestInfTrigger_TruncatesLongTitles(t *testing.T) {
	h := infNewHarness(infDroneSingle)
	hook := infHook()
	long := make([]rune, 2500)
	for i := range long {
		long[i] = 'é'
	}
	hook.Title = string(long)
	hook.Message = string(long)
	exec, err := h.trig.Trigger(context.Background(), infPipeline(), hook)
	require.NoError(t, err)
	require.Len(t, []rune(exec.Title), 2000)
	require.Len(t, []rune(exec.Message), 2000)
}

const infV1Pipeline = `version: 1
kind: pipeline
spec:
  stages:
  - name: build
    type: ci
    spec:
      steps:
      - name: compile
        type: script
        spec:
          run: go build
  - name: test
    type: ci
    spec:
      steps:
      - name: unit
        type: script
        spec:
          run: go test
`

func TestInfTrigger_V1YamlCreatesSequentialStages(t *testing.T) {
	h := infNewHarness(infV1Pipeline)
	exec, err := h.trig.Trigger(context.Background(), infPipeline(), infHook())
	require.NoError(t, err)
	require.NotNil(t, exec)

	require.Len(t, h.stages.created, 2)
	first, second := h.stages.created[0], h.stages.created[1]
	require.Equal(t, "build", first.Name)
	require.Equal(t, enum.CIStatusPending, first.Status)
	require.Equal(t, "test", second.Name)
	require.Equal(t, enum.CIStatusWaitingOnDeps, second.Status)
	require.Equal(t, []string{"build"}, second.DependsOn)
	require.True(t, first.OnSuccess)
	require.Len(t, h.sched.scheduled, 1)
}

func TestInfParseV1Stages_Errors(t *testing.T) {
	ctx := context.Background()
	repo := &types.Repository{ID: 1, Path: "space/repo", ParentID: 2}
	exec := &types.Execution{}

	tests := []struct {
		name    string
		data    string
		access  error
		wantErr string
	}{
		{name: "unparsable", data: "spec: [", wantErr: "could not parse v1 yaml"},
		{
			name: "non pipeline kind",
			data: "version: 1\nkind: template\ntype: stage\nspec:\n  stage:\n    type: ci\n    spec:\n      steps: []\n",
			// the exact wrapped message depends on the parser; any failure is acceptable here.
			wantErr: "",
		},
		{name: "public access failure", data: infV1Pipeline, access: errors.New("pa"), wantErr: "public access"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stages, err := parseV1Stages(ctx, []byte(tt.data), repo, exec, nil, nil, &infPublicAccess{err: tt.access})
			require.Error(t, err)
			require.Nil(t, stages)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func TestInfParseV1Stages_WhenEval(t *testing.T) {
	src := `version: 1
kind: pipeline
spec:
  stages:
  - name: only-main
    when: ${{ build.target == "main" }}
    type: ci
    spec:
      steps:
      - name: step
        type: script
        spec:
          run: echo
`
	repo := &types.Repository{ID: 1, Path: "space/repo", ParentID: 2}
	stages, err := parseV1Stages(context.Background(), []byte(src), repo, &types.Execution{Target: "main"},
		nil, nil, &infPublicAccess{})
	require.NoError(t, err)
	require.Len(t, stages, 1)
}

func TestInfIsV1Yaml(t *testing.T) {
	require.True(t, isV1Yaml([]byte(infV1Pipeline)))
	require.False(t, isV1Yaml([]byte(infDroneSingle)))
	require.False(t, isV1Yaml([]byte("  spec:\n")))
}

func TestInfTrunc(t *testing.T) {
	require.Equal(t, "abc", trunc("abc", 5))
	require.Equal(t, "ab", trunc("abc", 2))
	require.Equal(t, "éé", trunc("ééé", 2))
	require.Empty(t, trunc("", 3))
}

func TestInfCombineAndEnvs(t *testing.T) {
	require.Empty(t, combine())
	got := combine(map[string]string{"A": "1", "B": "1"}, nil, map[string]string{"B": "2"})
	require.Equal(t, map[string]string{"A": "1", "B": "2"}, got)

	env := Envs(context.Background(), &types.Repository{Path: "p/r"}, &types.Pipeline{Identifier: "id", Seq: 3},
		infURLProvider{})
	require.Equal(t, map[string]string{"DRONE_BUILD_LINK": "https://ui.invalid/p/r/id/3"}, env)
}
