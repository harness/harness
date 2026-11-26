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
	"net/http"
	"testing"

	"github.com/harness/gitness/app/api/controller/limiter"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// crpRepoCheck is a repo Check with injectable errors.
type crpRepoCheck struct {
	createErr    error
	lifecycleErr error
}

func (c crpRepoCheck) Create(context.Context, *auth.Session, *CheckInput) error { return c.createErr }

func (c crpRepoCheck) LifecycleRestriction(context.Context, *auth.Session, *types.RepositoryCore) error {
	return c.lifecycleErr
}

func crpDeletedRepo() *types.Repository {
	deleted := int64(12345)

	return &types.Repository{
		ID: crpRepoID, ParentID: 3, Identifier: "gone", Path: "space/gone", GitUID: "goneuid",
		State: enum.RepoStateActive, Deleted: &deleted,
	}
}

func TestCrpFind(t *testing.T) {
	ctx := context.Background()

	t.Run("returns the output with backfilled urls", func(t *testing.T) {
		e := crpNewEnv(t)
		e.public.public[crpRepoPath] = true
		e.c.favoriteStore = crpFavoriteStore{favorites: map[int64]bool{crpRepoID: true}}

		out, err := e.c.Find(ctx, crpSession(), crpRepoRef)

		require.NoError(t, err)
		assert.Equal(t, crpRepoID, out.ID)
		assert.Equal(t, "https://git.example.com/space/repo.git", out.GitURL)
		assert.Equal(t, "ssh://git.example.com/space/repo.git", out.GitSSHURL)
		assert.True(t, out.IsPublic)
		assert.True(t, out.IsFavorite)
		assert.False(t, out.Importing)
	})

	t.Run("works for repos being imported", func(t *testing.T) {
		e := crpNewEnv(t)
		e.setRepo(func(r *types.RepositoryCore) { r.State = enum.RepoStateGitImport })
		e.repoStore.repos[crpRepoID].State = enum.RepoStateGitImport

		out, err := e.c.Find(ctx, crpSession(), crpRepoRef)

		require.NoError(t, err)
		assert.True(t, out.Importing)
	})

	failures := []struct {
		name  string
		setup func(e *crpEnv)
		ref   string
		want  error
	}{
		{name: "unknown repo", ref: crpUnknownRepoRef, want: gitnessstore.ErrResourceNotFound},
		{name: "authorizer failure", ref: crpRepoRef, want: errCrpBoom, setup: func(e *crpEnv) {
			e.setAuthorizer(crpErrAuthorizer{err: errCrpBoom})
		}},
		{name: "store failure", ref: crpRepoRef, want: errCrpBoom, setup: func(e *crpEnv) {
			e.repoStore.errs["Find"] = errCrpBoom
		}},
		{name: "public access failure", ref: crpRepoRef, want: errCrpBoom, setup: func(e *crpEnv) {
			e.public.getErr = errCrpBoom
		}},
		{name: "favorites failure", ref: crpRepoRef, want: errCrpBoom, setup: func(e *crpEnv) {
			e.c.favoriteStore = crpFavoriteStore{err: errCrpBoom}
		}},
	}
	for _, tt := range failures {
		t.Run(tt.name, func(t *testing.T) {
			e := crpNewEnv(t)
			if tt.setup != nil {
				tt.setup(e)
			}

			_, err := e.c.Find(ctx, crpSession(), tt.ref)

			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestCrpGetRepoOutput(t *testing.T) {
	ctx := context.Background()

	t.Run("fork whose upstream is gone loses its fork id", func(t *testing.T) {
		e := crpNewEnv(t)
		repo := &types.Repository{ID: 1, Path: "space/fork", ForkID: 555, State: enum.RepoStateImportFailed}

		out, err := GetRepoOutput(ctx, e.c.publicAccess, e.c.repoFinder, repo)

		require.NoError(t, err)
		assert.Zero(t, out.ForkID)
		assert.Nil(t, out.Upstream)
		assert.True(t, out.ImportFailed)
		assert.EqualValues(t, 555, repo.ForkID, "input must not be modified")
	})

	t.Run("fork with upstream", func(t *testing.T) {
		e := crpNewEnv(t)
		repo := &types.Repository{ID: 1, Path: "space/fork", ForkID: crpRepoID, State: enum.RepoStateArchived}

		out, err := GetRepoOutput(ctx, e.c.publicAccess, e.c.repoFinder, repo)

		require.NoError(t, err)
		require.NotNil(t, out.Upstream)
		assert.Equal(t, crpRepoID, out.Upstream.ID)
		assert.True(t, out.Archived)
	})

	t.Run("public access failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.public.getErr = errCrpBoom

		_, err := GetRepoOutput(ctx, e.c.publicAccess, e.c.repoFinder, &types.Repository{})

		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("with access: upstream lookup failure is an error", func(t *testing.T) {
		e := crpNewEnv(t)

		_, err := GetRepoOutputWithAccess(ctx, e.c.repoFinder, true, &types.Repository{ForkID: 555})

		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
	})

	t.Run("with access: flags", func(t *testing.T) {
		e := crpNewEnv(t)

		out, err := GetRepoOutputWithAccess(ctx, e.c.repoFinder, true,
			&types.Repository{ForkID: crpRepoID, State: enum.RepoStateMigrateGitPush})

		require.NoError(t, err)
		assert.True(t, out.IsPublic)
		assert.True(t, out.Importing)
		require.NotNil(t, out.Upstream)
	})
}

func TestCrpUpdateDefaultBranch(t *testing.T) {
	ctx := context.Background()
	in := &UpdateDefaultBranchInput{Name: "develop"}

	t.Run("updates git and the database", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("UpdateDefaultBranch", mock.Anything, mock.MatchedBy(func(p *git.UpdateDefaultBranchParams) bool {
			return p.BranchName == "develop" && p.RepoUID == crpGitUID
		})).Return(nil)

		out, err := e.c.UpdateDefaultBranch(ctx, crpSession(), crpRepoRef, in)

		require.NoError(t, err)
		assert.Equal(t, "develop", out.DefaultBranch)
		assert.Equal(t, "develop", e.repoStore.repos[crpRepoID].DefaultBranch)
	})

	t.Run("access is checked", func(t *testing.T) {
		e := crpNewEnv(t)
		e.setAuthorizer(crpErrAuthorizer{err: errCrpBoom})

		_, err := e.c.UpdateDefaultBranch(ctx, crpSession(), crpRepoRef, in)

		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("store lookup failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.repoStore.errs["Find"] = errCrpBoom

		_, err := e.c.UpdateDefaultBranch(ctx, crpSession(), crpRepoRef, in)

		require.ErrorIs(t, err, errCrpBoom)
		e.git.AssertNotCalled(t, "UpdateDefaultBranch", mock.Anything, mock.Anything)
	})

	t.Run("git failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("UpdateDefaultBranch", mock.Anything, mock.Anything).Return(errCrpBoom)

		_, err := e.c.UpdateDefaultBranch(ctx, crpSession(), crpRepoRef, in)

		require.ErrorIs(t, err, errCrpBoom)
		assert.False(t, e.repoStore.called("UpdateOptLock"))
	})

	t.Run("database failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("UpdateDefaultBranch", mock.Anything, mock.Anything).Return(nil)
		e.repoStore.errs["UpdateOptLock"] = errCrpBoom

		_, err := e.c.UpdateDefaultBranch(ctx, crpSession(), crpRepoRef, in)

		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("output failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("UpdateDefaultBranch", mock.Anything, mock.Anything).Return(nil)
		e.public.getErr = errCrpBoom

		_, err := e.c.UpdateDefaultBranch(ctx, crpSession(), crpRepoRef, in)

		require.ErrorIs(t, err, errCrpBoom)
	})
}

func TestCrpSoftDelete(t *testing.T) {
	ctx := context.Background()

	t.Run("soft deletes and drops public access", func(t *testing.T) {
		e := crpNewEnv(t)
		e.public.public[crpRepoPath] = true

		out, err := e.c.SoftDelete(ctx, crpSession(), crpRepoRef)

		require.NoError(t, err)
		assert.NotZero(t, out.DeletedAt)
		assert.True(t, e.repoStore.called("SoftDelete"))
		assert.Equal(t, []string{crpRepoPath}, e.public.deleted)
	})

	failures := []struct {
		name  string
		setup func(e *crpEnv)
		ref   string
		check func(t *testing.T, err error)
	}{
		{name: "unknown repo", ref: crpUnknownRepoRef, check: func(t *testing.T, err error) {
			require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
		}},
		{name: "denied", ref: crpRepoRef, setup: func(e *crpEnv) { e.setAuthorizer(crpErrAuthorizer{err: errCrpBoom}) },
			check: func(t *testing.T, err error) { require.ErrorIs(t, err, errCrpBoom) }},
		{name: "lifecycle restriction", ref: crpRepoRef, setup: func(e *crpEnv) {
			e.c.repoCheck = crpRepoCheck{lifecycleErr: errCrpBoom}
		}, check: func(t *testing.T, err error) { require.ErrorIs(t, err, errCrpBoom) }},
		{name: "store lookup failure", ref: crpRepoRef, setup: func(e *crpEnv) {
			e.repoStore.errs["Find"] = errCrpBoom
		}, check: func(t *testing.T, err error) { require.ErrorIs(t, err, errCrpBoom) }},
		{name: "already deleted", ref: crpRepoRef, setup: func(e *crpEnv) {
			deleted := int64(1)
			e.repoStore.repos[crpRepoID].Deleted = &deleted
		}, check: func(t *testing.T, err error) {
			require.Error(t, err)
			assert.Equal(t, http.StatusBadRequest, crpUserErrStatus(t, err))
		}},
		{name: "public access lookup failure", ref: crpRepoRef, setup: func(e *crpEnv) {
			e.public.getErr = errCrpBoom
		}, check: func(t *testing.T, err error) { require.ErrorIs(t, err, errCrpBoom) }},
		{name: "public access delete failure", ref: crpRepoRef, setup: func(e *crpEnv) {
			e.public.deleteErr = errCrpBoom
		}, check: func(t *testing.T, err error) { require.ErrorIs(t, err, errCrpBoom) }},
		{name: "store failure", ref: crpRepoRef, setup: func(e *crpEnv) {
			e.repoStore.errs["SoftDelete"] = errCrpBoom
		}, check: func(t *testing.T, err error) { require.ErrorIs(t, err, errCrpBoom) }},
	}
	for _, tt := range failures {
		t.Run(tt.name, func(t *testing.T) {
			e := crpNewEnv(t)
			if tt.setup != nil {
				tt.setup(e)
			}

			_, err := e.c.SoftDelete(ctx, crpSession(), tt.ref)

			tt.check(t, err)
		})
	}

	t.Run("repositories in transient states are purged right away", func(t *testing.T) {
		e := crpNewEnv(t)
		e.repoStore.repos[crpRepoID].State = enum.RepoStateImportFailed
		e.git.On("DeleteRepository", mock.Anything, mock.Anything).Return(nil)

		_, err := e.c.SoftDelete(ctx, crpSession(), crpRepoRef)

		require.NoError(t, err)
		assert.True(t, e.repoStore.called("Purge"))
		assert.False(t, e.repoStore.called("SoftDelete"))
		assert.NotContains(t, e.repoStore.repos, crpRepoID)
	})
}

func TestCrpPurge(t *testing.T) {
	ctx := context.Background()

	newEnv := func(t *testing.T) *crpEnv {
		t.Helper()
		e := crpNewEnv(t)
		e.repoStore.repos[crpRepoID] = crpDeletedRepo()

		return e
	}

	t.Run("purges a deleted repository", func(t *testing.T) {
		e := newEnv(t)
		e.git.On("DeleteRepository", mock.Anything, mock.MatchedBy(func(p *git.DeleteRepositoryParams) bool {
			return p.RepoUID == "goneuid"
		})).Return(nil)

		err := e.c.Purge(ctx, crpSession(), crpRepoRef, 12345)

		require.NoError(t, err)
		assert.Equal(t, []string{"ClearForkID", "Purge"}, e.repoStore.calls[1:])
		e.git.AssertNumberOfCalls(t, "DeleteRepository", 1)
	})

	t.Run("fork counter of the upstream is decremented", func(t *testing.T) {
		e := newEnv(t)
		e.repoStore.repos[crpRepoID].ForkID = 99
		e.git.On("DeleteRepository", mock.Anything, mock.Anything).Return(nil)

		require.NoError(t, e.c.Purge(ctx, crpSession(), crpRepoRef, 12345))

		assert.True(t, e.repoStore.called("UpdateNumForks"))
	})

	t.Run("git failures do not fail the purge", func(t *testing.T) {
		e := newEnv(t)
		e.git.On("DeleteRepository", mock.Anything, mock.Anything).Return(errCrpBoom)

		require.NoError(t, e.c.Purge(ctx, crpSession(), crpRepoRef, 12345))
	})

	t.Run("missing git repository is ignored", func(t *testing.T) {
		e := newEnv(t)
		e.git.On("DeleteRepository", mock.Anything, mock.Anything).Return(errors.NotFound("gone"))

		require.NoError(t, e.c.DeleteGitRepository(ctx, crpSession(), "goneuid"))
	})

	t.Run("other git errors are reported", func(t *testing.T) {
		e := newEnv(t)
		e.git.On("DeleteRepository", mock.Anything, mock.Anything).Return(errCrpBoom)

		require.ErrorIs(t, e.c.DeleteGitRepository(ctx, crpSession(), "goneuid"), errCrpBoom)
	})

	t.Run("repository has to be deleted first", func(t *testing.T) {
		e := newEnv(t)
		e.repoStore.repos[crpRepoID].Deleted = nil

		err := e.c.Purge(ctx, crpSession(), crpRepoRef, 12345)

		require.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, crpUserErrStatus(t, err))
		assert.False(t, e.repoStore.called("Purge"))
	})

	failures := []struct {
		name  string
		setup func(e *crpEnv)
		want  error
	}{
		{name: "unknown deleted repo", setup: func(e *crpEnv) { delete(e.repoStore.repos, crpRepoID) },
			want: gitnessstore.ErrResourceNotFound},
		{name: "denied", setup: func(e *crpEnv) { e.setAuthorizer(crpErrAuthorizer{err: errCrpBoom}) }, want: errCrpBoom},
		{name: "lifecycle restriction", setup: func(e *crpEnv) {
			e.c.repoCheck = crpRepoCheck{lifecycleErr: errCrpBoom}
		}, want: errCrpBoom},
		{name: "clear fork failure", want: errCrpBoom, setup: func(e *crpEnv) {
			e.repoStore.errs["ClearForkID"] = errCrpBoom
		}},
		{name: "purge failure", setup: func(e *crpEnv) { e.repoStore.errs["Purge"] = errCrpBoom }, want: errCrpBoom},
	}
	for _, tt := range failures {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			tt.setup(e)

			err := e.c.Purge(ctx, crpSession(), crpRepoRef, 12345)

			require.ErrorIs(t, err, tt.want)
		})
	}

	t.Run("fork counter failure", func(t *testing.T) {
		e := newEnv(t)
		e.repoStore.repos[crpRepoID].ForkID = 99
		e.repoStore.errs["UpdateNumForks"] = errCrpBoom

		require.ErrorIs(t, e.c.Purge(ctx, crpSession(), crpRepoRef, 12345), errCrpBoom)
	})
}

func TestCrpEnsureIdentifierAvailable(t *testing.T) {
	ctx := context.Background()

	t.Run("free identifier", func(t *testing.T) {
		e := crpNewEnv(t)

		require.NoError(t, e.c.EnsureIdentifierAvailable(ctx, crpSession(), 3, crpOtherSpacePath))
	})

	t.Run("taken identifier is a conflict", func(t *testing.T) {
		e := crpNewEnv(t)

		err := e.c.EnsureIdentifierAvailable(ctx, crpSession(), 3, "repo")

		require.Error(t, err)
		assert.Equal(t, http.StatusConflict, crpUserErrStatus(t, err))
	})

	t.Run("store failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.repoStore.errs["FindActiveByUID"] = errCrpBoom

		require.ErrorIs(t, e.c.EnsureIdentifierAvailable(ctx, crpSession(), 3, "repo"), errCrpBoom)
	})

	t.Run("failed import is purged", func(t *testing.T) {
		e := crpNewEnv(t)
		e.repoStore.repos[crpRepoID].State = enum.RepoStateImportFailed
		e.git.On("DeleteRepository", mock.Anything, mock.Anything).Return(nil)

		require.NoError(t, e.c.EnsureIdentifierAvailable(ctx, crpSession(), 3, "repo"))

		assert.True(t, e.repoStore.called("Purge"))
		assert.Equal(t, []string{crpRepoPath}, e.public.deleted)
	})

	t.Run("failed import: public access delete failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.repoStore.repos[crpRepoID].State = enum.RepoStateImportFailed
		e.public.deleteErr = errCrpBoom

		require.ErrorIs(t, e.c.EnsureIdentifierAvailable(ctx, crpSession(), 3, "repo"), errCrpBoom)
	})

	t.Run("failed import: purge failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.repoStore.repos[crpRepoID].State = enum.RepoStateImportFailed
		e.repoStore.errs["Purge"] = errCrpBoom

		require.ErrorIs(t, e.c.EnsureIdentifierAvailable(ctx, crpSession(), 3, "repo"), errCrpBoom)
	})
}

func TestCrpRestore(t *testing.T) {
	ctx := context.Background()

	newEnv := func(t *testing.T) *crpEnv {
		t.Helper()
		e := crpNewEnv(t)
		e.repoStore.repos[crpRepoID] = crpDeletedRepo()

		return e
	}

	t.Run("restores in place", func(t *testing.T) {
		e := newEnv(t)

		out, err := e.c.Restore(ctx, crpSession(), crpRepoRef, 12345, &RestoreInput{})

		require.NoError(t, err)
		assert.Equal(t, "gone", out.Identifier)
		assert.False(t, out.IsPublic)
		assert.Nil(t, e.repoStore.restored.Deleted)
	})

	t.Run("restores with a new identifier and parent", func(t *testing.T) {
		e := newEnv(t)
		e.spaces.spaces[4] = &types.Space{ID: 4, Path: crpOtherSpacePath}
		newID, newParent := "renamed", crpOtherSpacePath

		out, err := e.c.Restore(ctx, crpSession(), crpRepoRef, 12345,
			&RestoreInput{NewIdentifier: &newID, NewParentRef: &newParent})

		require.NoError(t, err)
		assert.Equal(t, "renamed", out.Identifier)
		assert.EqualValues(t, 4, out.ParentID)
	})

	t.Run("unknown new parent is a bad request", func(t *testing.T) {
		e := newEnv(t)
		parent := "nowhere"

		_, err := e.c.Restore(ctx, crpSession(), crpRepoRef, 12345, &RestoreInput{NewParentRef: &parent})

		require.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, crpUserErrStatus(t, err))
	})

	t.Run("new parent lookup failure", func(t *testing.T) {
		e := newEnv(t)
		e.spaces.err = errCrpBoom
		parent := crpOtherSpacePath

		_, err := e.c.Restore(ctx, crpSession(), crpRepoRef, 12345, &RestoreInput{NewParentRef: &parent})

		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("not deleted", func(t *testing.T) {
		e := newEnv(t)
		e.repoStore.repos[crpRepoID].Deleted = nil

		_, err := e.c.Restore(ctx, crpSession(), crpRepoRef, 12345, &RestoreInput{})

		require.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, crpUserErrStatus(t, err))
	})

	t.Run("lookup, access and lifecycle failures", func(t *testing.T) {
		e := newEnv(t)
		delete(e.repoStore.repos, crpRepoID)
		_, err := e.c.Restore(ctx, crpSession(), crpRepoRef, 12345, &RestoreInput{})
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)

		e = newEnv(t)
		e.setAuthorizer(crpErrAuthorizer{err: errCrpBoom})
		_, err = e.c.Restore(ctx, crpSession(), crpRepoRef, 12345, &RestoreInput{})
		require.ErrorIs(t, err, errCrpBoom)

		e = newEnv(t)
		e.c.repoCheck = crpRepoCheck{lifecycleErr: errCrpBoom}
		_, err = e.c.Restore(ctx, crpSession(), crpRepoRef, 12345, &RestoreInput{})
		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("store failure", func(t *testing.T) {
		e := newEnv(t)
		e.repoStore.errs["Restore"] = errCrpBoom

		_, err := e.c.Restore(ctx, crpSession(), crpRepoRef, 12345, &RestoreInput{})

		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("repo quota reached", func(t *testing.T) {
		e := newEnv(t)
		e.c.resourceLimiter = crpRepoCountLimiter{err: errCrpBoom}

		_, err := e.c.Restore(ctx, crpSession(), crpRepoRef, 12345, &RestoreInput{})

		require.ErrorIs(t, err, limiter.ErrMaxNumReposReached)
	})

	t.Run("upstream lookup failure", func(t *testing.T) {
		e := newEnv(t)
		e.repoStore.repos[crpRepoID].ForkID = 555

		_, err := e.c.Restore(ctx, crpSession(), crpRepoRef, 12345, &RestoreInput{})

		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
	})
}

// crpRepoCountLimiter rejects repository creation.
type crpRepoCountLimiter struct {
	limiter.Unlimited
	err error
}

func (l crpRepoCountLimiter) RepoCount(context.Context, int64, int) error { return l.err }
