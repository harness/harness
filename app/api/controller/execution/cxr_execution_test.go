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

package execution

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/pipeline/commit"
	"github.com/harness/gitness/app/pipeline/triggerer"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

const (
	cxrRepoRef  = "root/repo"
	cxrPipeline = "pipe"
)

var errCxrBoom = errors.New("boom")

type cxrCanceler struct {
	err    error
	called int
}

func (c *cxrCanceler) Cancel(context.Context, *types.RepositoryCore, *types.Execution) error {
	c.called++
	return c.err
}

type cxrCommitService struct {
	commit.Service
	err error
}

func (c cxrCommitService) FindRef(_ context.Context, _ *types.RepositoryCore, ref string) (*types.Commit, error) {
	if c.err != nil {
		return nil, c.err
	}
	return &types.Commit{
		SHA:     sha.Must("0123456789abcdef0123456789abcdef01234567"),
		Title:   "title of " + ref,
		Message: "message",
		Author:  types.Signature{Identity: types.Identity{Name: "Jane", Email: "jane@example.com"}},
	}, nil
}

type cxrTriggerer struct {
	hook *triggerer.Hook
	err  error
}

func (t *cxrTriggerer) Trigger(
	_ context.Context, pipeline *types.Pipeline, hook *triggerer.Hook,
) (*types.Execution, error) {
	t.hook = hook
	if t.err != nil {
		return nil, t.err
	}
	return &types.Execution{PipelineID: pipeline.ID, Source: hook.Source, Status: enum.CIStatusPending}, nil
}

type cxrEnv struct {
	*cxrBase
	repo      *types.Repository
	pipelines store.PipelineStore
	execs     store.ExecutionStore
	stages    store.StageStore
	pipeline  *types.Pipeline
	canceler  *cxrCanceler
	trigger   *cxrTriggerer
	commitErr error
}

func cxrNewEnv(ctx context.Context, t *testing.T) *cxrEnv {
	t.Helper()
	b := cxrNewBase(ctx, t)
	repo := b.createRepo(ctx, t, b.createSpace(ctx, t, "root"), "repo")
	e := &cxrEnv{
		cxrBase:   b,
		repo:      repo,
		pipelines: database.NewPipelineStore(b.db),
		execs:     database.NewExecutionStore(b.db),
		stages:    database.NewStageStore(b.db),
		canceler:  &cxrCanceler{},
		trigger:   &cxrTriggerer{},
	}
	e.pipeline = &types.Pipeline{
		Identifier: cxrPipeline, RepoID: repo.ID, CreatedBy: b.admin.ID, ConfigPath: ".x.yml", DefaultBranch: "dev",
	}
	require.NoError(t, e.pipelines.Create(ctx, e.pipeline))

	exec := &types.Execution{
		PipelineID: e.pipeline.ID, RepoID: repo.ID, CreatedBy: b.admin.ID, Number: 1, Status: enum.CIStatusRunning,
	}
	require.NoError(t, e.execs.Create(ctx, exec))
	require.NoError(t, e.stages.Create(ctx, &types.Stage{
		ExecutionID: exec.ID, RepoID: repo.ID, Number: 1, Name: "build", Status: enum.CIStatusRunning,
	}))
	return e
}

func (e *cxrEnv) controller(mode cxrAuthMode) *Controller {
	pCache := cache.ProvidePrincipalInfoCache(database.NewPrincipalInfoView(e.db))
	return NewController(
		dbtx.New(e.db), cxrAuthorizer{mode: mode}, e.execs, database.NewCheckStore(e.db, pCache),
		e.canceler, cxrCommitService{err: e.commitErr}, e.trigger, e.stages, e.pipelines, e.repoFinder)
}

func TestCxrExecutionFind(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	c := e.controller(cxrAllow)

	got, err := c.Find(ctx, e.session(), cxrRepoRef, cxrPipeline, 1)
	require.NoError(t, err)
	require.EqualValues(t, 1, got.Number)
	require.Len(t, got.Stages, 1)
	require.Equal(t, "build", got.Stages[0].Name)

	_, err = c.Find(ctx, e.session(), cxrRepoRef, cxrPipeline, 99)
	require.ErrorContains(t, err, "failed to find execution 99")
	_, err = c.Find(ctx, e.session(), cxrRepoRef, "nope", 1)
	require.ErrorContains(t, err, "failed to find pipeline")
	_, err = c.Find(ctx, e.session(), "root/nope", cxrPipeline, 1)
	require.ErrorContains(t, err, "failed to find repo")
	_, err = e.controller(cxrDeny).Find(ctx, e.session(), cxrRepoRef, cxrPipeline, 1)
	require.ErrorContains(t, err, "failed to authorize")
}

func TestCxrExecutionList(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	c := e.controller(cxrAllow)

	list, count, err := c.List(ctx, e.session(), cxrRepoRef, cxrPipeline, types.Pagination{Page: 1, Size: 10})
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.Len(t, list, 1)

	_, _, err = c.List(ctx, e.session(), cxrRepoRef, "nope", types.Pagination{})
	require.ErrorContains(t, err, "failed to find pipeline")
	_, _, err = e.controller(cxrDeny).List(ctx, e.session(), cxrRepoRef, cxrPipeline, types.Pagination{})
	require.ErrorContains(t, err, "failed to authorize")
}

func TestCxrExecutionDelete(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	c := e.controller(cxrAllow)

	require.ErrorContains(t, e.controller(cxrDeny).Delete(ctx, e.session(), cxrRepoRef, cxrPipeline, 1),
		"failed to authorize")
	require.ErrorContains(t, c.Delete(ctx, e.session(), cxrRepoRef, "nope", 1), "failed to find pipeline")
	require.NoError(t, c.Delete(ctx, e.session(), cxrRepoRef, cxrPipeline, 1))
	_, err := c.Find(ctx, e.session(), cxrRepoRef, cxrPipeline, 1)
	require.ErrorContains(t, err, "failed to find execution")
	// Deleting a missing execution is a no-op in the store.
	require.NoError(t, c.Delete(ctx, e.session(), cxrRepoRef, cxrPipeline, 1))
}

func TestCxrExecutionCancel(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	c := e.controller(cxrAllow)

	got, err := c.Cancel(ctx, e.session(), cxrRepoRef, cxrPipeline, 1)
	require.NoError(t, err)
	require.EqualValues(t, 1, got.Number)
	require.Equal(t, 1, e.canceler.called)

	_, err = c.Cancel(ctx, e.session(), cxrRepoRef, cxrPipeline, 99)
	require.ErrorContains(t, err, "failed to find execution")
	_, err = c.Cancel(ctx, e.session(), cxrRepoRef, "nope", 1)
	require.ErrorContains(t, err, "failed to find pipeline")
	_, err = e.controller(cxrDeny).Cancel(ctx, e.session(), cxrRepoRef, cxrPipeline, 1)
	require.ErrorContains(t, err, "failed to authorize")

	e.canceler.err = errCxrBoom
	_, err = c.Cancel(ctx, e.session(), cxrRepoRef, cxrPipeline, 1)
	require.ErrorIs(t, err, errCxrBoom)
}

func TestCxrExecutionCreate(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	c := e.controller(cxrAllow)

	// An empty branch falls back to the pipeline default branch.
	got, err := c.Create(ctx, e.session(), cxrRepoRef, cxrPipeline, "")
	require.NoError(t, err)
	require.Equal(t, "dev", got.Source)
	require.Equal(t, "refs/heads/dev", e.trigger.hook.Ref)
	require.Equal(t, "Jane", e.trigger.hook.AuthorName)
	require.Equal(t, e.admin.UID, e.trigger.hook.Trigger)

	got, err = c.Create(ctx, e.session(), cxrRepoRef, cxrPipeline, "feature")
	require.NoError(t, err)
	require.Equal(t, "feature", got.Source)

	// Without pipeline default branch the repo default branch is used.
	_, err = e.db.ExecContext(ctx, "UPDATE pipelines SET pipeline_default_branch = '' WHERE pipeline_id = $1",
		e.pipeline.ID)
	require.NoError(t, err)
	got, err = c.Create(ctx, e.session(), cxrRepoRef, cxrPipeline, "")
	require.NoError(t, err)
	require.Equal(t, "main", got.Source)

	_, err = c.Create(ctx, e.session(), cxrRepoRef, "nope", "")
	require.ErrorContains(t, err, "failed to find pipeline")
	_, err = e.controller(cxrDeny).Create(ctx, e.session(), cxrRepoRef, cxrPipeline, "")
	require.ErrorContains(t, err, "failed to authorize")

	e.commitErr = errCxrBoom
	_, err = e.controller(cxrAllow).Create(ctx, e.session(), cxrRepoRef, cxrPipeline, "")
	require.ErrorIs(t, err, errCxrBoom)

	e.commitErr = nil
	e.trigger.err = errCxrBoom
	_, err = e.controller(cxrAllow).Create(ctx, e.session(), cxrRepoRef, cxrPipeline, "")
	require.ErrorIs(t, err, errCxrBoom)
}
