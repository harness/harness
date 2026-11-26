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
	"sync"
	"testing"

	"github.com/harness/gitness/app/api/controller/service"
	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/pipeline/scheduler"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/livelog"
	mocksse "github.com/harness/gitness/mocks/sse"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// infPrincipalStore creates the service principal on demand.
type infPrincipalStore struct {
	store.PrincipalStore

	services map[string]*types.Service
}

func (s *infPrincipalStore) FindServiceByUID(_ context.Context, uid string) (*types.Service, error) {
	svc, ok := s.services[uid]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return svc, nil
}

func (s *infPrincipalStore) CreateService(_ context.Context, svc *types.Service) error {
	svc.ID = 77
	s.services[svc.UID] = svc
	return nil
}

var infBootstrapOnce sync.Once

// infBootstrapPipelineService initialises the (process wide) pipeline service principal.
func infBootstrapPipelineService(t *testing.T) {
	t.Helper()
	var err error
	infBootstrapOnce.Do(func() {
		st := &infPrincipalStore{services: map[string]*types.Service{}}
		ctrl := service.NewController(check.PrincipalUIDDefault, nil, st)
		cfg := &types.Config{}
		cfg.Principal.Pipeline.UID = "pipe-svc"
		cfg.Principal.Pipeline.Email = "pipeline@example.invalid"
		cfg.Principal.Pipeline.DisplayName = "Pipeline"
		err = bootstrap.PipelineService(context.Background(), cfg, ctrl)
	})
	require.NoError(t, err)
}

type infManagerHarness struct {
	executions *infExecutionStore
	stages     *infStageStore
	steps      *infStepStore
	repos      *infRepoStore
	pipelines  *infPipelineStore
	checks     *infCheckStore
	secrets    *infSecretStore
	logs       *infLogStore
	logz       *infLogStream
	sched      *infScheduler
	files      *infFileService
	conv       *infConverter
	access     *infPublicAccess
	sse        *mocksse.Streamer
	mgr        *Manager
}

func infNewManagerHarness() *infManagerHarness {
	h := &infManagerHarness{
		executions: &infExecutionStore{},
		stages:     &infStageStore{},
		steps:      &infStepStore{},
		repos: &infRepoStore{repo: &types.Repository{
			ID: 3, Path: "space/repo", Identifier: "repo", ParentID: 2, GitUID: "git-uid",
		}},
		pipelines: &infPipelineStore{pipeline: &types.Pipeline{ID: 5, ConfigPath: ".harness/ci.yaml", Identifier: "ci"}},
		checks:    &infCheckStore{},
		secrets:   &infSecretStore{},
		logs:      &infLogStore{},
		logz:      &infLogStream{},
		sched:     &infScheduler{},
		files:     &infFileService{data: "kind: pipeline"},
		conv:      &infConverter{},
		access:    &infPublicAccess{},
		sse:       &mocksse.Streamer{},
	}
	h.sse.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return()
	h.mgr = New(&types.Config{}, h.executions, h.pipelines, infURLProvider{}, h.sse, h.files, h.conv,
		h.logs, h.logz, h.checks, h.repos, h.sched, h.secrets, h.stages, h.steps, nil, h.access, infReporter())
	return h
}

// wireDetails makes stage 1 / execution 2 resolvable.
func (h *infManagerHarness) wireDetails(execution *types.Execution) {
	h.stages.findFn = func(id int64) (*types.Stage, error) {
		return &types.Stage{ID: id, ExecutionID: 2, Name: "build"}, nil
	}
	h.stages.listFn = func(int64) ([]*types.Stage, error) { return []*types.Stage{{ID: 1}}, nil }
	h.executions.findFn = func(int64) (*types.Execution, error) { return execution, nil }
}

func TestInfManager_Request(t *testing.T) {
	ctx := context.Background()
	h := infNewManagerHarness()
	var got scheduler.Filter
	h.sched.requestFn = func(f scheduler.Filter) (*types.Stage, error) {
		got = f
		return &types.Stage{ID: 4}, nil
	}
	stage, err := h.mgr.Request(ctx, &Request{
		Kind: "pipeline", Type: "docker", OS: "linux", Arch: "amd64", Variant: "v8", Kernel: "5",
		Labels: map[string]string{"a": "b"},
	})
	require.NoError(t, err)
	require.EqualValues(t, 4, stage.ID)
	require.Equal(t, scheduler.Filter{
		Kind: "pipeline", Type: "docker", OS: "linux", Arch: "amd64", Variant: "v8", Kernel: "5",
		Labels: map[string]string{"a": "b"},
	}, got)

	h.sched.requestFn = func(scheduler.Filter) (*types.Stage, error) { return nil, errInfBoom }
	_, err = h.mgr.Request(ctx, &Request{})
	require.ErrorIs(t, err, errInfBoom)

	canceled, cancel := context.WithCancel(ctx)
	cancel()
	h.sched.requestFn = func(scheduler.Filter) (*types.Stage, error) { return nil, context.Canceled }
	_, err = h.mgr.Request(canceled, &Request{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestInfManager_Accept(t *testing.T) {
	ctx := context.Background()

	t.Run("assigns the machine", func(t *testing.T) {
		h := infNewManagerHarness()
		h.stages.findFn = func(id int64) (*types.Stage, error) { return &types.Stage{ID: id}, nil }
		stage, err := h.mgr.Accept(ctx, 1, "machine-1")
		require.NoError(t, err)
		require.Equal(t, "machine-1", stage.Machine)
		require.Equal(t, enum.CIStatusPending, stage.Status)
		require.Len(t, h.stages.updated, 1)
	})
	t.Run("already assigned", func(t *testing.T) {
		h := infNewManagerHarness()
		h.stages.findFn = func(id int64) (*types.Stage, error) { return &types.Stage{ID: id, Machine: "other"}, nil }
		_, err := h.mgr.Accept(ctx, 1, "machine-1")
		require.ErrorContains(t, err, "stage already assigned")
		require.Empty(t, h.stages.updated)
	})
	t.Run("not found", func(t *testing.T) {
		h := infNewManagerHarness()
		h.stages.findFn = func(int64) (*types.Stage, error) { return nil, errInfBoom }
		_, err := h.mgr.Accept(ctx, 1, "m")
		require.ErrorIs(t, err, errInfBoom)
	})
	t.Run("update conflicts are returned with the stage", func(t *testing.T) {
		for _, updateErr := range []error{gitnessstore.ErrVersionConflict, errInfBoom} {
			h := infNewManagerHarness()
			h.stages.findFn = func(id int64) (*types.Stage, error) { return &types.Stage{ID: id}, nil }
			h.stages.updateFn = func(*types.Stage) error { return updateErr }
			stage, err := h.mgr.Accept(ctx, 1, "m")
			require.ErrorIs(t, err, updateErr)
			require.NotNil(t, stage)
		}
	})
}

func TestInfManager_WriteAndUploadLogs(t *testing.T) {
	ctx := context.Background()
	h := infNewManagerHarness()

	line := &livelog.Line{Number: 1, Message: "hi"}
	require.NoError(t, h.mgr.Write(ctx, 5, line))
	require.Equal(t, []*livelog.Line{line}, h.logz.lines)
	h.logz.writeErr = errInfBoom
	require.ErrorIs(t, h.mgr.Write(ctx, 5, line), errInfBoom)

	require.NoError(t, h.mgr.UploadLogs(ctx, 5, strings.NewReader("full logs")))
	require.Equal(t, "full logs", h.logs.uploads[5])
	h.logs.err = errInfBoom
	require.ErrorIs(t, h.mgr.UploadLogs(ctx, 5, strings.NewReader("x")), errInfBoom)
}

func TestInfManager_Details(t *testing.T) {
	infBootstrapPipelineService(t)
	ctx := context.Background()

	t.Run("collects everything the runner needs", func(t *testing.T) {
		h := infNewManagerHarness()
		h.access.public = true
		h.wireDetails(&types.Execution{ID: 2, RepoID: 3, PipelineID: 5, After: "abc", Event: enum.TriggerEventPush})
		h.secrets.secrets = []*types.Secret{{Identifier: "token", Data: "value"}}

		details, err := h.mgr.Details(ctx, 1)
		require.NoError(t, err)
		require.EqualValues(t, 1, details.Stage.ID)
		require.True(t, details.RepoIsPublic)
		require.Equal(t, "https://git.example.invalid/space/repo.git", details.Repo.GitURL)
		require.Len(t, details.Execution.Stages, 1)
		require.Equal(t, "kind: pipeline", string(details.Config.Data))
		require.Equal(t, "value", details.Secrets[0].Data)
		require.Equal(t, "git.example.invalid", details.Netrc.Machine)
		require.NotEmpty(t, details.Netrc.Login)
		require.Equal(t, 3, strings.Count(details.Netrc.Password, ".")+1, "password is a JWT")
	})
	t.Run("pull requests do not receive secret values", func(t *testing.T) {
		h := infNewManagerHarness()
		h.wireDetails(&types.Execution{ID: 2, RepoID: 3, PipelineID: 5, Event: enum.TriggerEventPullRequest})
		h.secrets.secrets = []*types.Secret{{Identifier: "token", Data: "value"}}
		details, err := h.mgr.Details(ctx, 1)
		require.NoError(t, err)
		require.Equal(t, "token", details.Secrets[0].Identifier)
		require.Empty(t, details.Secrets[0].Data)
	})

	failures := map[string]func(h *infManagerHarness){
		"stage not found": func(h *infManagerHarness) {
			h.stages.findFn = func(int64) (*types.Stage, error) { return nil, errInfBoom }
		},
		"execution not found": func(h *infManagerHarness) {
			h.executions.findFn = func(int64) (*types.Execution, error) { return nil, errInfBoom }
		},
		"pipeline not found": func(h *infManagerHarness) { h.pipelines.err = errInfBoom },
		"repo not found":     func(h *infManagerHarness) { h.repos.err = errInfBoom },
		"stage list fails": func(h *infManagerHarness) {
			h.stages.listFn = func(int64) ([]*types.Stage, error) { return nil, errInfBoom }
		},
		"secrets fail":       func(h *infManagerHarness) { h.secrets.err = errInfBoom },
		"file fetch fails":   func(h *infManagerHarness) { h.files.err = errInfBoom },
		"public access fail": func(h *infManagerHarness) { h.access.err = errInfBoom },
		"conversion fails":   func(h *infManagerHarness) { h.conv.err = errInfBoom },
	}
	for name, mutate := range failures {
		t.Run(name, func(t *testing.T) {
			h := infNewManagerHarness()
			h.wireDetails(&types.Execution{ID: 2, RepoID: 3, PipelineID: 5})
			mutate(h)
			_, err := h.mgr.Details(ctx, 1)
			require.ErrorIs(t, err, errInfBoom)
		})
	}
}

func TestInfManager_CreateNetrc_InvalidCloneURL(t *testing.T) {
	infBootstrapPipelineService(t)
	h := infNewManagerHarness()
	_, err := h.mgr.createNetrc(&types.Repository{GitURL: "://bad url"})
	require.ErrorContains(t, err, "failed to parse clone url")
}

func TestInfManager_StepHooks(t *testing.T) {
	ctx := context.Background()

	wire := func(h *infManagerHarness) {
		h.stages.findFn = func(int64) (*types.Stage, error) { return &types.Stage{ID: 1, ExecutionID: 2}, nil }
		h.executions.findFn = func(int64) (*types.Execution, error) { return &types.Execution{ID: 2, RepoID: 3}, nil }
		h.stages.listWithStepsFn = func(int64) ([]*types.Stage, error) { return []*types.Stage{{ID: 1}}, nil }
	}

	t.Run("BeforeStep creates the log stream and publishes the update", func(t *testing.T) {
		h := infNewManagerHarness()
		wire(h)
		require.NoError(t, h.mgr.BeforeStep(ctx, &types.Step{ID: 9, StageID: 1}))
		require.Equal(t, []int64{9}, h.logz.created)
		require.Len(t, h.steps.updated, 1)
		h.sse.AssertCalled(t, "Publish", int64(2), enum.SSETypeExecutionUpdated, mock.Anything)
	})
	t.Run("BeforeStep log stream failure", func(t *testing.T) {
		h := infNewManagerHarness()
		wire(h)
		h.logz.createErr = errInfBoom
		require.ErrorIs(t, h.mgr.BeforeStep(ctx, &types.Step{ID: 9}), errInfBoom)
		require.Empty(t, h.steps.updated)
	})
	t.Run("AfterStep deletes the log stream", func(t *testing.T) {
		h := infNewManagerHarness()
		wire(h)
		require.NoError(t, h.mgr.AfterStep(ctx, &types.Step{ID: 9, StageID: 1}))
		require.Equal(t, []int64{9}, h.logz.deleted)
	})
	t.Run("AfterStep returns the step update error and tolerates log errors", func(t *testing.T) {
		h := infNewManagerHarness()
		wire(h)
		h.steps.updateFn = func(*types.Step) error { return errInfBoom }
		h.logz.deleteErr = errInfBoom
		require.ErrorIs(t, h.mgr.AfterStep(ctx, &types.Step{ID: 9}), errInfBoom)

		h.steps.updateFn = nil
		h.logz.deleteErr = livelog.ErrStreamNotFound
		require.NoError(t, h.mgr.AfterStep(ctx, &types.Step{ID: 9, StageID: 1}))
	})
}

func TestInfUpdater(t *testing.T) {
	ctx := context.Background()
	long := strings.Repeat("e", 600)

	newUpdater := func(h *infManagerHarness) *updater {
		return &updater{
			Executions: h.executions, Repos: h.repos, SSEStreamer: h.sse, Steps: h.steps, Stages: h.stages,
		}
	}

	t.Run("truncates errors and publishes", func(t *testing.T) {
		h := infNewManagerHarness()
		h.stages.findFn = func(int64) (*types.Stage, error) { return &types.Stage{ExecutionID: 2}, nil }
		h.executions.findFn = func(int64) (*types.Execution, error) { return &types.Execution{RepoID: 3}, nil }
		h.stages.listWithStepsFn = func(int64) ([]*types.Stage, error) { return []*types.Stage{{ID: 1}}, nil }
		step := &types.Step{Error: long}
		require.NoError(t, newUpdater(h).do(ctx, step))
		require.Len(t, step.Error, 500)
		h.sse.AssertNumberOfCalls(t, "Publish", 1)
	})
	t.Run("step update failure is returned", func(t *testing.T) {
		h := infNewManagerHarness()
		h.steps.updateFn = func(*types.Step) error { return errInfBoom }
		require.ErrorIs(t, newUpdater(h).do(ctx, &types.Step{}), errInfBoom)
		h.sse.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything, mock.Anything)
	})
	t.Run("lookup failures after the update are swallowed", func(t *testing.T) {
		cases := map[string]func(h *infManagerHarness){
			"stage": func(h *infManagerHarness) {
				h.stages.findFn = func(int64) (*types.Stage, error) { return nil, errInfBoom }
			},
			"execution": func(h *infManagerHarness) {
				h.stages.findFn = func(int64) (*types.Stage, error) { return &types.Stage{}, nil }
				h.executions.findFn = func(int64) (*types.Execution, error) { return nil, errInfBoom }
			},
			"repo": func(h *infManagerHarness) {
				h.stages.findFn = func(int64) (*types.Stage, error) { return &types.Stage{}, nil }
				h.executions.findFn = func(int64) (*types.Execution, error) { return &types.Execution{}, nil }
				h.repos.err = errInfBoom
			},
			"stages": func(h *infManagerHarness) {
				h.stages.findFn = func(int64) (*types.Stage, error) { return &types.Stage{}, nil }
				h.executions.findFn = func(int64) (*types.Execution, error) { return &types.Execution{}, nil }
				h.stages.listWithStepsFn = func(int64) ([]*types.Stage, error) { return nil, errInfBoom }
			},
		}
		for name, mutate := range cases {
			h := infNewManagerHarness()
			mutate(h)
			require.NoError(t, newUpdater(h).do(ctx, &types.Step{}), name)
			h.sse.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything, mock.Anything)
		}
	})
}

func TestInfManager_Watch(t *testing.T) {
	ctx := context.Background()

	t.Run("cancellation errors are returned", func(t *testing.T) {
		h := infNewManagerHarness()
		h.sched.cancelledFn = func(int64) (bool, error) { return false, context.DeadlineExceeded }
		_, err := h.mgr.Watch(ctx, 1)
		require.ErrorIs(t, err, context.DeadlineExceeded)
	})
	t.Run("reports completion from the stored status", func(t *testing.T) {
		h := infNewManagerHarness()
		h.sched.cancelledFn = func(int64) (bool, error) { return false, nil }
		h.executions.findFn = func(int64) (*types.Execution, error) {
			return &types.Execution{Status: enum.CIStatusSuccess}, nil
		}
		done, err := h.mgr.Watch(ctx, 1)
		require.NoError(t, err)
		require.True(t, done)

		h.executions.findFn = func(int64) (*types.Execution, error) {
			return &types.Execution{Status: enum.CIStatusRunning}, nil
		}
		done, err = h.mgr.Watch(ctx, 1)
		require.NoError(t, err)
		require.False(t, done)
	})
	t.Run("execution lookup failure", func(t *testing.T) {
		h := infNewManagerHarness()
		h.sched.cancelledFn = func(int64) (bool, error) { return true, nil }
		h.executions.findFn = func(int64) (*types.Execution, error) { return nil, errInfBoom }
		ok, err := h.mgr.Watch(ctx, 1)
		require.ErrorContains(t, err, "could not find build for cancellation")
		require.True(t, ok)
	})
}

func TestInfManager_BeforeStage(t *testing.T) {
	ctx := context.Background()
	execution := func(status enum.CIStatus) *types.Execution {
		return &types.Execution{ID: 2, RepoID: 3, PipelineID: 5, Status: status}
	}
	wire := func(h *infManagerHarness, e *types.Execution) {
		h.executions.findFn = func(int64) (*types.Execution, error) { return e, nil }
		h.stages.listWithStepsFn = func(int64) ([]*types.Stage, error) { return []*types.Stage{{ID: 1}}, nil }
	}

	t.Run("marks a pending execution as running", func(t *testing.T) {
		h := infNewManagerHarness()
		e := execution(enum.CIStatusPending)
		wire(h, e)
		stage := &types.Stage{ID: 1, ExecutionID: 2, Error: strings.Repeat("x", 700),
			Steps: []*types.Step{{ID: 7, Error: strings.Repeat("y", 700)}}}
		require.NoError(t, h.mgr.BeforeStage(ctx, stage))

		require.Equal(t, enum.CIStatusRunning, e.Status)
		require.NotZero(t, e.Started)
		require.Len(t, stage.Error, 500)
		require.Len(t, h.steps.created, 1)
		require.Len(t, h.steps.created[0].Error, 500)
		require.Len(t, h.checks.upserted, 1)
		h.sse.AssertCalled(t, "Publish", int64(2), enum.SSETypeExecutionRunning, mock.Anything)
	})
	t.Run("running executions are left alone", func(t *testing.T) {
		h := infNewManagerHarness()
		e := execution(enum.CIStatusRunning)
		wire(h, e)
		updates := 0
		h.executions.updateFn = func(*types.Execution) error { updates++; return nil }
		require.NoError(t, h.mgr.BeforeStage(ctx, &types.Stage{ExecutionID: 2}))
		require.Zero(t, updates)
	})
	t.Run("version conflicts on the execution are ignored", func(t *testing.T) {
		h := infNewManagerHarness()
		wire(h, execution(enum.CIStatusPending))
		h.executions.updateFn = func(*types.Execution) error { return gitnessstore.ErrVersionConflict }
		require.NoError(t, h.mgr.BeforeStage(ctx, &types.Stage{ExecutionID: 2}))
	})
	t.Run("check store failures are tolerated", func(t *testing.T) {
		h := infNewManagerHarness()
		wire(h, execution(enum.CIStatusRunning))
		h.checks.err = errInfBoom
		require.NoError(t, h.mgr.BeforeStage(ctx, &types.Stage{ExecutionID: 2}))
	})

	failures := map[string]func(h *infManagerHarness){
		"execution not found": func(h *infManagerHarness) {
			h.executions.findFn = func(int64) (*types.Execution, error) { return nil, errInfBoom }
		},
		"repo not found": func(h *infManagerHarness) { h.repos.err = errInfBoom },
		"stage update":   func(h *infManagerHarness) { h.stages.updateFn = func(*types.Stage) error { return errInfBoom } },
		"step create":    func(h *infManagerHarness) { h.steps.createFn = func(*types.Step) error { return errInfBoom } },
		"execution update": func(h *infManagerHarness) {
			h.executions.updateFn = func(*types.Execution) error { return errInfBoom }
		},
		"pipeline lookup": func(h *infManagerHarness) { h.pipelines.err = errInfBoom },
		"stages with steps": func(h *infManagerHarness) {
			h.stages.listWithStepsFn = func(int64) ([]*types.Stage, error) { return nil, errInfBoom }
		},
	}
	for name, mutate := range failures {
		t.Run(name, func(t *testing.T) {
			h := infNewManagerHarness()
			wire(h, execution(enum.CIStatusPending))
			mutate(h)
			stage := &types.Stage{ExecutionID: 2, Steps: []*types.Step{{ID: 1}}}
			require.ErrorIs(t, h.mgr.BeforeStage(ctx, stage), errInfBoom)
		})
	}
}
