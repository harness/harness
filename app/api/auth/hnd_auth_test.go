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

package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/store"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

var errHndAuthz = errors.New("hnd authz failure")

type hndAuthMode int

const (
	hndAuthAllow hndAuthMode = iota
	hndAuthDeny
	hndAuthError
)

type hndAuthorizer struct {
	mode hndAuthMode
}

func (a hndAuthorizer) decide() (bool, error) {
	switch a.mode {
	case hndAuthAllow:
		return true, nil
	case hndAuthError:
		return false, errHndAuthz
	case hndAuthDeny:
		return false, nil
	default:
		return false, nil
	}
}

func (a hndAuthorizer) Check(
	context.Context, *auth.Session, *types.Scope, *types.Resource, enum.Permission,
) (bool, error) {
	return a.decide()
}

func (a hndAuthorizer) CheckAll(context.Context, *auth.Session, ...types.PermissionCheck) (bool, error) {
	return a.decide()
}

func (a hndAuthorizer) CheckMany(
	_ context.Context, _ *auth.Session, checks ...types.PermissionCheck,
) ([]bool, error) {
	ok, err := a.decide()
	res := make([]bool, len(checks))
	for i := range res {
		res[i] = ok
	}
	return res, err
}

func hndSession(anonymous bool) *auth.Session {
	if anonymous {
		return &auth.Session{Principal: auth.AnonymousPrincipal}
	}
	return &auth.Session{Principal: types.Principal{ID: 1, UID: "u1", Type: enum.PrincipalTypeUser}}
}

type hndSpaceStore struct {
	store.SpaceStore
	space *types.Space
	err   error
}

func (s hndSpaceStore) Find(context.Context, int64) (*types.Space, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.space, nil
}

type hndRepoStore struct {
	store.RepoStore
	repo *types.Repository
	err  error
}

func (s hndRepoStore) Find(context.Context, int64) (*types.Repository, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.repo, nil
}

func TestHndCheck(t *testing.T) {
	ctx := t.Context()
	scope := &types.Scope{SpacePath: "root"}
	resource := &types.Resource{Type: enum.ResourceTypeRepo, Identifier: "r1"}

	t.Run("allow", func(t *testing.T) {
		err := Check(ctx, hndAuthorizer{mode: hndAuthAllow}, hndSession(false),
			scope, resource, enum.PermissionRepoView)
		require.NoError(t, err)
	})

	t.Run("deny authenticated", func(t *testing.T) {
		err := Check(ctx, hndAuthorizer{mode: hndAuthDeny}, hndSession(false),
			scope, resource, enum.PermissionRepoView)
		require.ErrorIs(t, err, ErrForbidden)
	})

	t.Run("deny anonymous", func(t *testing.T) {
		err := Check(ctx, hndAuthorizer{mode: hndAuthDeny}, hndSession(true),
			scope, resource, enum.PermissionRepoView)
		require.ErrorIs(t, err, ErrUnauthorized)
	})

	t.Run("authorizer error", func(t *testing.T) {
		err := Check(ctx, hndAuthorizer{mode: hndAuthError}, hndSession(false),
			scope, resource, enum.PermissionRepoView)
		require.ErrorIs(t, err, errHndAuthz)
	})
}

func TestHndCheckAll(t *testing.T) {
	ctx := t.Context()
	checks := []types.PermissionCheck{{Permission: enum.PermissionRepoView}}

	err := CheckAll(ctx, hndAuthorizer{mode: hndAuthAllow}, hndSession(false), checks...)
	require.NoError(t, err)

	err = CheckAll(ctx, hndAuthorizer{mode: hndAuthDeny}, hndSession(true), checks...)
	require.ErrorIs(t, err, ErrUnauthorized)
}

func TestHndCheckSessionAuth(t *testing.T) {
	require.NoError(t, CheckSessionAuth(hndSession(false), true))
	require.ErrorIs(t, CheckSessionAuth(hndSession(false), false), ErrForbidden)
	require.ErrorIs(t, CheckSessionAuth(hndSession(true), false), ErrUnauthorized)
}

func TestHndIsNoAccess(t *testing.T) {
	require.True(t, IsNoAccess(ErrForbidden))
	require.True(t, IsNoAccess(ErrUnauthorized))
	require.False(t, IsNoAccess(errHndAuthz))
}

func TestHndCheckChild(t *testing.T) {
	ctx := t.Context()
	authz := hndAuthorizer{mode: hndAuthAllow}
	session := hndSession(false)

	t.Run("space parent", func(t *testing.T) {
		space := &types.Space{ID: 2, Path: "root/child", Identifier: "child"}
		err := CheckChild(ctx, authz, session, hndSpaceStore{space: space}, hndRepoStore{},
			enum.ParentResourceTypeSpace, 2, enum.ResourceTypeSecret, "s1", enum.PermissionSecretView)
		require.NoError(t, err)
	})

	t.Run("repo parent", func(t *testing.T) {
		repo := &types.Repository{ID: 3, Path: "root/repo1", Identifier: "repo1"}
		err := CheckChild(ctx, authz, session, hndSpaceStore{}, hndRepoStore{repo: repo},
			enum.ParentResourceTypeRepo, 3, enum.ResourceTypePipeline, "p1", enum.PermissionPipelineView)
		require.NoError(t, err)
	})

	t.Run("space not found", func(t *testing.T) {
		err := CheckChild(ctx, authz, session, hndSpaceStore{err: gitnessstore.ErrResourceNotFound}, hndRepoStore{},
			enum.ParentResourceTypeSpace, 99, enum.ResourceTypeSecret, "s1", enum.PermissionSecretView)
		require.Error(t, err)
	})

	t.Run("unknown parent", func(t *testing.T) {
		err := CheckChild(ctx, authz, session, hndSpaceStore{}, hndRepoStore{},
			enum.ParentResourceType("unknown"), 1, enum.ResourceTypeSecret, "s1", enum.PermissionSecretView)
		require.ErrorIs(t, err, ErrParentResourceTypeUnknown)
	})
}

func TestHndCheckRepo(t *testing.T) {
	ctx := t.Context()
	repo := &types.RepositoryCore{Path: "root/myrepo", State: enum.RepoStateActive}

	require.NoError(t, CheckRepo(ctx, hndAuthorizer{mode: hndAuthAllow}, hndSession(false),
		repo, enum.PermissionRepoView))

	err := CheckRepo(ctx, hndAuthorizer{mode: hndAuthDeny}, hndSession(false), repo, enum.PermissionRepoEdit)
	require.ErrorIs(t, err, ErrForbidden)

	ok, err := IsRepoOwner(ctx, hndAuthorizer{mode: hndAuthAllow}, hndSession(false), repo)
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = IsRepoOwner(ctx, hndAuthorizer{mode: hndAuthDeny}, hndSession(false), repo)
	require.NoError(t, err)
	require.False(t, ok)
}

func TestHndCheckRepoState(t *testing.T) {
	ctx := t.Context()
	session := hndSession(false)
	active := &types.RepositoryCore{State: enum.RepoStateActive}
	archived := &types.RepositoryCore{State: enum.RepoStateArchived}

	require.NoError(t, CheckRepoState(ctx, session, active, enum.PermissionRepoView))
	require.NoError(t, CheckRepoState(ctx, session, archived, enum.PermissionRepoView))
	require.Error(t, CheckRepoState(ctx, session, archived, enum.PermissionRepoEdit))

	// Additional allowed state skips default map lookup.
	importing := &types.RepositoryCore{State: enum.RepoStateGitImport}
	require.NoError(t, CheckRepoState(ctx, session, importing, enum.PermissionRepoView, enum.RepoStateGitImport))
}

func TestHndCheckSpace(t *testing.T) {
	ctx := t.Context()
	space := &types.SpaceCore{Path: "root/team"}

	require.NoError(t, CheckSpace(ctx, hndAuthorizer{mode: hndAuthAllow}, hndSession(false),
		space, enum.PermissionSpaceView))
	require.NoError(t, CheckSpaceScope(ctx, hndAuthorizer{mode: hndAuthAllow}, hndSession(false),
		space, enum.ResourceTypeSecret, enum.PermissionSecretView))
}

func TestHndCheckUser(t *testing.T) {
	ctx := t.Context()
	user := &types.User{UID: "alice"}
	require.NoError(t, CheckUser(ctx, hndAuthorizer{mode: hndAuthAllow}, hndSession(false), user, enum.PermissionUserEdit))
}

func TestHndCheckRepoBadPath(t *testing.T) {
	ctx := t.Context()
	repo := &types.RepositoryCore{Path: ""}
	err := CheckRepo(ctx, hndAuthorizer{mode: hndAuthAllow}, hndSession(false), repo, enum.PermissionRepoView)
	require.Error(t, err)
}

func TestHndCheckSpaceBadPath(t *testing.T) {
	ctx := t.Context()
	space := &types.SpaceCore{Path: ""}
	err := CheckSpace(ctx, hndAuthorizer{mode: hndAuthAllow}, hndSession(false), space, enum.PermissionSpaceView)
	require.Error(t, err)
}
