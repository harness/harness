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

package repo

import (
	"context"
	"testing"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// crpPipelineStore is a store.PipelineStore returning preconfigured pipelines.
type crpPipelineStore struct {
	store.PipelineStore
	pipelines []*types.Pipeline
	latest    []*types.Pipeline
	count     int64
	countErr  error
	listErr   error
	latestErr error
}

func (s crpPipelineStore) Count(context.Context, int64, *types.ListPipelinesFilter) (int64, error) {
	return s.count, s.countErr
}

func (s crpPipelineStore) List(context.Context, int64, *types.ListPipelinesFilter) ([]*types.Pipeline, error) {
	return s.pipelines, s.listErr
}

func (s crpPipelineStore) ListLatest(
	context.Context, int64, *types.ListPipelinesFilter,
) ([]*types.Pipeline, error) {
	return s.latest, s.latestErr
}

// crpExecutionStore is a store.ExecutionStore returning preconfigured executions per pipeline.
type crpExecutionStore struct {
	store.ExecutionStore
	execs map[int64][]*types.ExecutionInfo
	err   error
}

func (s crpExecutionStore) ListByPipelineIDs(
	context.Context, []int64, int64,
) (map[int64][]*types.ExecutionInfo, error) {
	return s.execs, s.err
}

func TestCrpListPipelines(t *testing.T) {
	ctx := context.Background()
	execs := map[int64][]*types.ExecutionInfo{1: {{Number: 7}}}

	newEnv := func(t *testing.T, ps crpPipelineStore) *crpEnv {
		t.Helper()
		e := crpNewEnv(t)
		e.c.pipelineStore = ps
		e.c.executionStore = crpExecutionStore{execs: execs}

		return e
	}

	t.Run("lists pipelines with their last executions", func(t *testing.T) {
		e := newEnv(t, crpPipelineStore{pipelines: []*types.Pipeline{{ID: 1}, {ID: 2}}, count: 2})

		out, count, err := e.c.ListPipelines(ctx, crpSession(), crpRepoRef, &types.ListPipelinesFilter{})

		require.NoError(t, err)
		assert.EqualValues(t, 2, count)
		require.Len(t, out, 2)
		require.Len(t, out[0].LastExecutions, 1)
		assert.EqualValues(t, 7, out[0].LastExecutions[0].Number)
		assert.Empty(t, out[1].LastExecutions)
	})

	t.Run("latest pipelines use the dedicated query", func(t *testing.T) {
		e := newEnv(t, crpPipelineStore{latest: []*types.Pipeline{{ID: 1}}, count: 1})

		out, _, err := e.c.ListPipelines(ctx, crpSession(), crpRepoRef, &types.ListPipelinesFilter{Latest: true})

		require.NoError(t, err)
		require.Len(t, out, 1)
	})

	failures := []struct {
		name   string
		store  crpPipelineStore
		exec   *crpExecutionStore
		filter types.ListPipelinesFilter
	}{
		{name: "count failure", store: crpPipelineStore{countErr: errCrpBoom}},
		{name: "list failure", store: crpPipelineStore{listErr: errCrpBoom}},
		{
			name:   "latest failure",
			store:  crpPipelineStore{latestErr: errCrpBoom},
			filter: types.ListPipelinesFilter{Latest: true},
		},
		{name: "executions failure", exec: &crpExecutionStore{err: errCrpBoom}},
	}
	for _, tt := range failures {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t, tt.store)
			if tt.exec != nil {
				e.c.executionStore = *tt.exec
			}

			_, _, err := e.c.ListPipelines(ctx, crpSession(), crpRepoRef, &tt.filter)

			require.ErrorIs(t, err, errCrpBoom)
		})
	}

	t.Run("access failures", func(t *testing.T) {
		e := newEnv(t, crpPipelineStore{})

		_, _, err := e.c.ListPipelines(ctx, crpSession(), "", &types.ListPipelinesFilter{})
		require.Error(t, err)

		_, _, err = e.c.ListPipelines(ctx, crpSession(), crpUnknownRepoRef, &types.ListPipelinesFilter{})
		require.Error(t, err)

		e.setRepo(func(r *types.RepositoryCore) { r.State = enum.RepoStateGitImport })
		_, _, err = e.c.ListPipelines(ctx, crpSession(), crpRepoRef, &types.ListPipelinesFilter{})
		require.ErrorContains(t, err, "not allowed for repository in state")

		e.setRepo(func(r *types.RepositoryCore) { r.State = enum.RepoStateActive })
		e.setAuthorizer(crpErrAuthorizer{err: errCrpBoom})
		_, _, err = e.c.ListPipelines(ctx, crpSession(), crpRepoRef, &types.ListPipelinesFilter{})
		require.ErrorIs(t, err, errCrpBoom)
	})
}

// crpPrincipalStore is a store.PrincipalStore serving service accounts.
type crpPrincipalStore struct {
	store.PrincipalStore
	accounts []*types.ServiceAccount
	count    int64
	listErr  error
	countErr error
	gotInfos []*types.ServiceAccountParentInfo
}

func (s *crpPrincipalStore) ListServiceAccounts(
	_ context.Context, infos []*types.ServiceAccountParentInfo, _ *types.PrincipalFilter,
) ([]*types.ServiceAccount, error) {
	s.gotInfos = infos

	return s.accounts, s.listErr
}

func (s *crpPrincipalStore) CountServiceAccounts(
	context.Context, []*types.ServiceAccountParentInfo, *types.PrincipalFilter,
) (int64, error) {
	return s.count, s.countErr
}

func TestCrpListServiceAccounts(t *testing.T) {
	ctx := context.Background()
	accounts := []*types.ServiceAccount{{ID: 1, UID: "sa1"}, {ID: 2, UID: "sa2"}}

	t.Run("lists the accounts of the repository only", func(t *testing.T) {
		e := crpNewEnv(t)
		principals := &crpPrincipalStore{accounts: accounts, count: 99}
		e.c.principalStore = principals

		out, count, err := e.c.ListServiceAccounts(
			ctx, crpSession(), crpRepoRef, false, &types.PrincipalFilter{Page: 1, Size: 10})

		require.NoError(t, err)
		assert.EqualValues(t, 2, count, "a short first page doesn't need a count query")
		require.Len(t, out, 2)
		assert.Equal(t, "sa1", out[0].UID)
		assert.Equal(t, []*types.ServiceAccountParentInfo{{ID: crpRepoID, Type: enum.ParentResourceTypeRepo}},
			principals.gotInfos)
	})

	t.Run("inherited accounts include the ancestor spaces", func(t *testing.T) {
		e := crpNewEnv(t)
		e.spaces.ancestors = []int64{10, 3}
		principals := &crpPrincipalStore{accounts: accounts, count: 99}
		e.c.principalStore = principals

		_, count, err := e.c.ListServiceAccounts(
			ctx, crpSession(), crpRepoRef, true, &types.PrincipalFilter{Page: 2, Size: 2})

		require.NoError(t, err)
		assert.EqualValues(t, 99, count)
		require.Len(t, principals.gotInfos, 3)
		assert.Equal(t, enum.ParentResourceTypeSpace, principals.gotInfos[0].Type)
		assert.EqualValues(t, 10, principals.gotInfos[0].ID)
		assert.Equal(t, enum.ParentResourceTypeRepo, principals.gotInfos[2].Type)
	})

	failures := []struct {
		name      string
		setup     func(e *crpEnv, p *crpPrincipalStore)
		inherited bool
	}{
		{name: "no permission", setup: func(e *crpEnv, _ *crpPrincipalStore) {
			e.setAuthorizer(crpErrAuthorizer{err: errCrpBoom})
		}},
		{name: "ancestor lookup failure", inherited: true, setup: func(e *crpEnv, _ *crpPrincipalStore) {
			e.spaces.ancestorErr = errCrpBoom
		}},
		{name: "list failure", setup: func(_ *crpEnv, p *crpPrincipalStore) { p.listErr = errCrpBoom }},
		{name: "count failure", setup: func(_ *crpEnv, p *crpPrincipalStore) {
			p.accounts = accounts
			p.countErr = errCrpBoom
		}},
	}
	for _, tt := range failures {
		t.Run(tt.name, func(t *testing.T) {
			e := crpNewEnv(t)
			principals := &crpPrincipalStore{}
			e.c.principalStore = principals
			tt.setup(e, principals)

			_, _, err := e.c.ListServiceAccounts(
				ctx, crpSession(), crpRepoRef, tt.inherited, &types.PrincipalFilter{Page: 2, Size: 1})

			require.ErrorIs(t, err, errCrpBoom)
		})
	}
}
