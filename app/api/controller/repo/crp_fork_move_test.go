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
	"encoding/json"
	"testing"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/controller/limiter"
	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/hook"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// crpForkCreateEnv prepares an environment able to run CreateFork end to end.
const crpMissingBranch = "missing"

func crpForkCreateEnv(t *testing.T) (*crpEnv, crpSettingsStore) {
	t.Helper()

	e := crpNewEnv(t)
	e.withImportJobs(t, crpJobStore{})
	settingsStore := crpSettingsStore{values: map[string]json.RawMessage{}}
	e.c.settings = settings.NewService(settingsStore, e.c.spaceFinder)
	e.git.On("CreateRepository", mock.Anything, mock.Anything).Return(&git.CreateRepositoryOutput{UID: "forkuid"}, nil)

	return e, settingsStore
}

func TestCrpCreateFork(t *testing.T) {
	ctx := context.Background()
	in := func() *CreateForkInput { return &CreateForkInput{ParentRef: "3", Identifier: "myfork"} }

	t.Run("forks the repository", func(t *testing.T) {
		e, settingsStore := crpForkCreateEnv(t)

		out, err := e.c.CreateFork(ctx, crpSession(), crpRepoRef, in())

		require.NoError(t, err)
		assert.Equal(t, "myfork", out.Identifier)
		assert.Equal(t, crpRepoID, out.ForkID)
		assert.Equal(t, enum.RepoStateGitImport, out.State)
		assert.True(t, out.Importing)
		assert.False(t, out.IsPublic)
		assert.Equal(t, crpDefaultBranch, out.DefaultBranch)
		assert.True(t, e.repoStore.called("UpdateNumForks"))
		assert.JSONEq(t, "false", string(settingsStore.values[string(settings.KeyGitLFSEnabled)]))
		require.NotNil(t, out.Upstream)
		assert.Equal(t, crpRepoID, out.Upstream.ID)
	})

	t.Run("fork branch becomes the default branch", func(t *testing.T) {
		e, _ := crpForkCreateEnv(t)
		e.git.On("GetBranch", mock.Anything, mock.MatchedBy(func(p *git.GetBranchParams) bool {
			return p.BranchName == crpReleaseBranch
		})).Return(&git.GetBranchOutput{Branch: crpBranch(crpReleaseBranch, crpTestSHA)}, nil)
		input := in()
		input.ForkBranch = crpReleaseBranch

		out, err := e.c.CreateFork(ctx, crpSession(), crpRepoRef, input)

		require.NoError(t, err)
		assert.Equal(t, crpReleaseBranch, out.DefaultBranch)
	})

	t.Run("the default branch needs no lookup", func(t *testing.T) {
		e, _ := crpForkCreateEnv(t)
		input := in()
		input.ForkBranch = crpDefaultBranch

		_, err := e.c.CreateFork(ctx, crpSession(), crpRepoRef, input)

		require.NoError(t, err)
		e.git.AssertNotCalled(t, "GetBranch", mock.Anything, mock.Anything)
	})

	t.Run("settings failure is tolerated", func(t *testing.T) {
		e, _ := crpForkCreateEnv(t)
		e.c.settings = settings.NewService(
			crpSettingsStore{values: map[string]json.RawMessage{}, upsertErr: errCrpBoom}, e.c.spaceFinder)

		_, err := e.c.CreateFork(ctx, crpSession(), crpRepoRef, in())

		require.NoError(t, err)
	})

	t.Run("reference sync failure is tolerated", func(t *testing.T) {
		e, _ := crpForkCreateEnv(t)
		e.withImportJobs(t, crpJobStore{createErr: errCrpBoom})

		_, err := e.c.CreateFork(ctx, crpSession(), crpRepoRef, in())

		require.NoError(t, err)
	})

	failures := []struct {
		name  string
		setup func(e *crpEnv)
		in    func() *CreateForkInput
		ref   string
		check func(t *testing.T, e *crpEnv, err error)
	}{
		{
			name: "linked repositories can't be forked",
			setup: func(e *crpEnv) {
				e.setRepo(func(r *types.RepositoryCore) { r.Type = enum.RepoTypeLinked })
			},
			check: func(t *testing.T, _ *crpEnv, err error) { require.ErrorContains(t, err, "can't be forked") },
		},
		{
			name:  "missing parent",
			in:    func() *CreateForkInput { return &CreateForkInput{Identifier: "x"} },
			check: func(t *testing.T, _ *crpEnv, err error) { require.True(t, errors.IsInvalidArgument(err)) },
		},
		{
			name:  "missing identifier",
			in:    func() *CreateForkInput { return &CreateForkInput{ParentRef: "3"} },
			check: func(t *testing.T, _ *crpEnv, err error) { require.True(t, errors.IsInvalidArgument(err)) },
		},
		{
			name:  "upstream lookup failure",
			setup: func(e *crpEnv) { e.repoStore.errs["Find"] = errCrpBoom },
			check: func(t *testing.T, _ *crpEnv, err error) { require.ErrorIs(t, err, errCrpBoom) },
		},
		{
			name:  "empty repositories can't be forked",
			setup: func(e *crpEnv) { e.repoStore.repos[crpRepoID].IsEmpty = true },
			check: func(t *testing.T, _ *crpEnv, err error) { require.True(t, errors.IsInvalidArgument(err)) },
		},
		{
			name: "no permission to create in the parent",
			setup: func(e *crpEnv) {
				e.setAuthorizer(crpDenyPermAuthorizer{denied: map[enum.Permission]bool{enum.PermissionRepoCreate: true}})
			},
			check: func(t *testing.T, _ *crpEnv, err error) { require.Error(t, err) },
		},
		{
			name: "fork branch not found",
			setup: func(e *crpEnv) {
				e.git.On("GetBranch", mock.Anything, mock.Anything).Return(nil, errors.NotFound("branch"))
			},
			in: func() *CreateForkInput {
				input := in()
				input.ForkBranch = crpMissingBranch

				return input
			},
			check: func(t *testing.T, _ *crpEnv, err error) { require.True(t, errors.IsInvalidArgument(err)) },
		},
		{
			name: "fork branch lookup failure",
			setup: func(e *crpEnv) {
				e.git.On("GetBranch", mock.Anything, mock.Anything).Return(nil, errCrpBoom)
			},
			in: func() *CreateForkInput {
				input := in()
				input.ForkBranch = crpMissingBranch

				return input
			},
			check: func(t *testing.T, _ *crpEnv, err error) { require.ErrorIs(t, err, errCrpBoom) },
		},
		{
			name:  "repo check rejects",
			setup: func(e *crpEnv) { e.c.repoCheck = crpRepoCheck{createErr: errCrpBoom} },
			check: func(t *testing.T, _ *crpEnv, err error) { require.ErrorIs(t, err, errCrpBoom) },
		},
		{
			name: "git failure",
			setup: func(e *crpEnv) {
				e.git.ExpectedCalls = nil
				e.git.On("CreateRepository", mock.Anything, mock.Anything).Return(nil, errCrpBoom)
			},
			check: func(t *testing.T, e *crpEnv, err error) {
				require.ErrorIs(t, err, errCrpBoom)
				assert.False(t, e.repoStore.called("Create"))
			},
		},
		{
			name: "repo quota reached",
			setup: func(e *crpEnv) {
				e.git.On("DeleteRepository", mock.Anything, mock.Anything).Return(nil)
				e.c.resourceLimiter = crpRepoCountLimiter{err: errCrpBoom}
			},
			check: func(t *testing.T, _ *crpEnv, err error) { require.ErrorIs(t, err, limiter.ErrMaxNumReposReached) },
		},
		{
			name: "storage over limit",
			setup: func(e *crpEnv) {
				e.git.On("DeleteRepository", mock.Anything, mock.Anything).Return(nil)
				e.c.resourceLimiter = restoreLimiterStub{
					rootSpaceStorageErr: &limiter.TotalQuotaStorageError{Level: limiter.TotalQuotaLevelOverLimit},
				}
			},
			check: func(t *testing.T, _ *crpEnv, err error) { require.Error(t, err) },
		},
		{
			name: "space lookup failure",
			setup: func(e *crpEnv) {
				e.git.On("DeleteRepository", mock.Anything, mock.Anything).Return(nil)
				e.spaces.err = errCrpBoom
			},
			check: func(t *testing.T, _ *crpEnv, err error) { require.ErrorIs(t, err, errCrpBoom) },
		},
		{
			name: "create failure",
			setup: func(e *crpEnv) {
				e.git.On("DeleteRepository", mock.Anything, mock.Anything).Return(errCrpBoom)
				e.repoStore.errs["Create"] = errCrpBoom
			},
			check: func(t *testing.T, _ *crpEnv, err error) { require.ErrorIs(t, err, errCrpBoom) },
		},
		{
			name: "fork counter failure",
			setup: func(e *crpEnv) {
				e.git.On("DeleteRepository", mock.Anything, mock.Anything).Return(nil)
				e.repoStore.errs["UpdateNumForks"] = errCrpBoom
			},
			check: func(t *testing.T, _ *crpEnv, err error) { require.ErrorIs(t, err, errCrpBoom) },
		},
		{
			name: "public access failure rolls back",
			setup: func(e *crpEnv) {
				e.git.On("DeleteRepository", mock.Anything, mock.Anything).Return(nil)
				e.public.setErr = errCrpBoom
			},
			check: func(t *testing.T, e *crpEnv, err error) {
				require.ErrorIs(t, err, errCrpBoom)
				assert.True(t, e.repoStore.called("Purge"))
			},
		},
		{
			name: "public access failure and failed cleanup",
			setup: func(e *crpEnv) {
				e.public.setErr = errCrpBoom
				e.public.deleteErr = errCrpBoom
			},
			check: func(t *testing.T, e *crpEnv, err error) {
				require.ErrorIs(t, err, errCrpBoom)
				assert.False(t, e.repoStore.called("Purge"))
			},
		},
		{
			name: "public access failure and failed purge",
			setup: func(e *crpEnv) {
				e.public.setErr = errCrpBoom
				e.repoStore.errs["Purge"] = errCrpBoom
			},
			check: func(t *testing.T, _ *crpEnv, err error) { require.ErrorIs(t, err, errCrpBoom) },
		},
	}
	for _, tt := range failures {
		t.Run(tt.name, func(t *testing.T) {
			e, _ := crpForkCreateEnv(t)
			if tt.setup != nil {
				tt.setup(e)
			}
			input := in()
			if tt.in != nil {
				input = tt.in()
			}

			_, err := e.c.CreateFork(ctx, crpSession(), crpRepoRef, input)

			tt.check(t, e, err)
		})
	}
}

// crpConnectorService is an importer.ConnectorService with a fixed access info result.
type crpConnectorService struct {
	importer.ConnectorService
	info         importer.AccessInfo
	err          error
	providerInfo importer.ProviderRepoInfo
	providerErr  error
}

func (s crpConnectorService) FetchProviderRepoInfo(
	context.Context, importer.ConnectorDef,
) (importer.ProviderRepoInfo, error) {
	return s.providerInfo, s.providerErr
}

func (crpConnectorService) ResolveConnectorRef(parentSpacePath, ref string) (string, string) {
	return parentSpacePath, ref
}

func (s crpConnectorService) GetAccessInfo(context.Context, importer.ConnectorDef) (importer.AccessInfo, error) {
	return s.info, s.err
}

// crpLinkedRepoStore is a store.LinkedRepoStore returning a fixed linked repository.
type crpLinkedRepoStore struct {
	store.LinkedRepoStore
	linked *types.LinkedRepo
	err    error
}

func (s crpLinkedRepoStore) Find(context.Context, int64) (*types.LinkedRepo, error) {
	return s.linked, s.err
}

func TestCrpLinkedSyncInput_Sanitize(t *testing.T) {
	tests := []struct {
		name     string
		branches []string
		want     []string
		wantErr  bool
	}{
		{name: "empty", wantErr: true},
		{name: "blank branch", branches: []string{"  "}, wantErr: true},
		{name: "invalid characters", branches: []string{"a b"}, wantErr: true},
		{name: "wildcard", branches: []string{"a*"}, wantErr: true},
		{name: "colon", branches: []string{"a:b"}, wantErr: true},
		{name: "sorted and deduplicated", branches: []string{" b ", "a", "b"}, want: []string{"a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := &LinkedSyncInput{Branches: tt.branches}

			err := in.sanitize()

			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, errors.IsInvalidArgument(err))
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, in.Branches)
		})
	}
}

func TestCrpLinkedSync(t *testing.T) {
	ctx := context.Background()
	linked := &types.LinkedRepo{ConnectorPath: "acc", ConnectorIdentifier: "conn", ConnectorRepo: "owner/name"}
	access := importer.AccessInfo{URL: "https://github.example.com/owner/name.git", Username: "u", Password: "p"}

	newEnv := func(t *testing.T) *crpEnv {
		t.Helper()
		e := crpNewEnv(t)
		e.setRepo(func(r *types.RepositoryCore) { r.Type = enum.RepoTypeLinked })
		e.c.linkedRepoStore = crpLinkedRepoStore{linked: linked}
		e.c.connectorService = crpConnectorService{info: access}

		return e
	}
	in := func() *LinkedSyncInput { return &LinkedSyncInput{Branches: []string{crpDefaultBranch, "dev"}} }

	t.Run("syncs the branches and indexes the default branch", func(t *testing.T) {
		e := newEnv(t)
		indexer := &crpIndexer{}
		e.c.indexer = indexer
		newSHA := crpTestSHA
		e.git.On("SyncRefs", mock.Anything, mock.MatchedBy(func(p *git.SyncRefsParams) bool {
			return p.Source == "https://u:p@github.example.com/owner/name.git" &&
				len(p.Refs) == 2 && p.Refs[0] == "refs/heads/dev" && p.Refs[1] == "refs/heads/main"
		})).Return(&git.SyncRefsOutput{Refs: []hook.ReferenceUpdate{{Ref: "refs/heads/main", New: newSHA}}}, nil)

		out, err := e.c.LinkedSync(ctx, crpSession(), crpRepoRef, in())

		require.NoError(t, err)
		require.Len(t, out.Refs, 1)
		assert.Equal(t, "refs/heads/main", out.Refs[0].Ref)
		assert.Equal(t, []int64{crpRepoID}, indexer.indexed)
	})

	t.Run("other branches are not indexed", func(t *testing.T) {
		e := newEnv(t)
		indexer := &crpIndexer{}
		e.c.indexer = indexer
		e.git.On("SyncRefs", mock.Anything, mock.Anything).Return(
			&git.SyncRefsOutput{Refs: []hook.ReferenceUpdate{{Ref: "refs/heads/dev"}}}, nil)

		_, err := e.c.LinkedSync(ctx, crpSession(), crpRepoRef, in())

		require.NoError(t, err)
		assert.Empty(t, indexer.indexed)
	})

	t.Run("indexing failure is tolerated", func(t *testing.T) {
		e := newEnv(t)
		e.c.indexer = &crpIndexer{err: errCrpBoom}
		e.git.On("SyncRefs", mock.Anything, mock.Anything).Return(
			&git.SyncRefsOutput{Refs: []hook.ReferenceUpdate{{Ref: "refs/heads/main"}}}, nil)

		_, err := e.c.LinkedSync(ctx, crpSession(), crpRepoRef, in())

		require.NoError(t, err)
	})

	failures := []struct {
		name  string
		setup func(e *crpEnv)
		in    *LinkedSyncInput
		check func(t *testing.T, err error)
	}{
		{name: "invalid input", in: &LinkedSyncInput{}, check: func(t *testing.T, err error) {
			require.True(t, errors.IsInvalidArgument(err))
		}},
		{name: "not a linked repository", setup: func(e *crpEnv) {
			e.setRepo(func(r *types.RepositoryCore) { r.Type = enum.RepoTypeNormal })
		}, check: func(t *testing.T, err error) { require.True(t, errors.IsInvalidArgument(err)) }},
		{name: "linked repo lookup failure", setup: func(e *crpEnv) {
			e.c.linkedRepoStore = crpLinkedRepoStore{err: errCrpBoom}
		}, check: func(t *testing.T, err error) { require.ErrorIs(t, err, errCrpBoom) }},
		{name: "connector failure", setup: func(e *crpEnv) {
			e.c.connectorService = crpConnectorService{err: errCrpBoom}
		}, check: func(t *testing.T, err error) { require.ErrorIs(t, err, errCrpBoom) }},
		{name: "access info without usable url", setup: func(e *crpEnv) {
			e.c.connectorService = crpConnectorService{info: importer.AccessInfo{URL: "://bad"}}
		}, check: func(t *testing.T, err error) { require.True(t, errors.IsInvalidArgument(err)) }},
		{name: "git failure", setup: func(e *crpEnv) {
			e.git.On("SyncRefs", mock.Anything, mock.Anything).Return(nil, errCrpBoom)
		}, check: func(t *testing.T, err error) { require.ErrorIs(t, err, errCrpBoom) }},
		{name: "repo reload failure after default branch update", setup: func(e *crpEnv) {
			e.repoStore.errs["Find"] = errCrpBoom
			e.git.On("SyncRefs", mock.Anything, mock.Anything).Return(
				&git.SyncRefsOutput{Refs: []hook.ReferenceUpdate{{Ref: "refs/heads/main"}}}, nil)
		}, check: func(t *testing.T, err error) { require.ErrorIs(t, err, errCrpBoom) }},
	}
	for _, tt := range failures {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			if tt.setup != nil {
				tt.setup(e)
			}
			input := in()
			if tt.in != nil {
				input = tt.in
			}

			_, err := e.c.LinkedSync(ctx, crpSession(), crpRepoRef, input)

			tt.check(t, err)
		})
	}
}

func TestCrpLinkedSync_UnknownRepository(t *testing.T) {
	e := crpNewEnv(t)

	_, err := e.c.LinkedSync(context.Background(), crpSession(), crpUnknownRepoRef,
		&LinkedSyncInput{Branches: []string{crpDefaultBranch}})

	require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
}

func TestCrpLinkedSync_NeedsPush(t *testing.T) {
	e := crpNewEnv(t)
	e.setRepo(func(r *types.RepositoryCore) { r.Type = enum.RepoTypeLinked })
	e.setAuthorizer(crpDenyPermAuthorizer{denied: map[enum.Permission]bool{enum.PermissionRepoPush: true}})

	in := &LinkedSyncInput{Branches: []string{crpDefaultBranch}}

	_, err := e.c.LinkedSync(context.Background(), crpSession(), crpRepoRef, in)

	require.ErrorIs(t, err, apiauth.ErrForbidden)
}
