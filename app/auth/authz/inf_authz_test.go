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

package authz

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/jwt"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	appcache "github.com/harness/gitness/app/store/cache"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

// infMapCache is a trivial cache.Cache backed by a function.
type infMapCache[K comparable, V any] struct {
	get func(key K) (V, error)
}

func (c infMapCache[K, V]) Stats() (int64, int64) { return 0, 0 }

func (c infMapCache[K, V]) Get(_ context.Context, key K) (V, error) { return c.get(key) }

func (c infMapCache[K, V]) Evict(context.Context, K) {}

type infSpaces struct {
	byID   map[int64]*types.SpaceCore
	byPath map[string]*types.SpaceCore
	errID  error
}

func infNewSpaces(spaces ...*types.SpaceCore) *infSpaces {
	s := &infSpaces{byID: map[int64]*types.SpaceCore{}, byPath: map[string]*types.SpaceCore{}}
	for _, sp := range spaces {
		s.byID[sp.ID] = sp
		s.byPath[sp.Path] = sp
	}
	return s
}

func (s *infSpaces) finder() refcache.SpaceFinder {
	idCache := infMapCache[int64, *types.SpaceCore]{get: func(id int64) (*types.SpaceCore, error) {
		if s.errID != nil {
			return nil, s.errID
		}
		sp, ok := s.byID[id]
		if !ok {
			return nil, gitnessstore.ErrResourceNotFound
		}
		return sp, nil
	}}
	pathCache := infMapCache[string, *types.SpacePath]{get: func(path string) (*types.SpacePath, error) {
		sp, ok := s.byPath[path]
		if !ok {
			return nil, gitnessstore.ErrResourceNotFound
		}
		return &types.SpacePath{Value: path, SpaceID: sp.ID}, nil
	}}
	ciCache := infMapCache[string, int64]{get: func(string) (int64, error) { return 0, gitnessstore.ErrResourceNotFound }}
	return refcache.NewSpaceFinder(idCache, pathCache, ciCache, appcache.Evictor[*types.SpaceCore]{})
}

type infMembershipStore struct {
	store.MembershipStore

	memberships map[types.MembershipKey]*types.Membership
	err         error
}

func (s *infMembershipStore) Find(_ context.Context, key types.MembershipKey) (*types.Membership, error) {
	if s.err != nil {
		return nil, s.err
	}
	m, ok := s.memberships[key]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return m, nil
}

type infPublicAccess struct {
	publicaccess.Service

	public map[string]bool
	err    error
	calls  []string
}

func (p *infPublicAccess) Get(_ context.Context, t enum.PublicResourceType, path string) (bool, error) {
	p.calls = append(p.calls, fmt.Sprintf("%s:%s", t, path))
	if p.err != nil {
		return false, p.err
	}
	return p.public[fmt.Sprintf("%s:%s", t, path)], nil
}

func TestInfCheckPublicAccess(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name     string
		resource types.Resource
		scope    types.Scope
		perm     enum.Permission
		public   map[string]bool
		wantKey  string
		want     bool
	}{
		{
			name:     "public space view",
			resource: types.Resource{Type: enum.ResourceTypeSpace, Identifier: "child"},
			scope:    types.Scope{SpacePath: "root"},
			perm:     enum.PermissionSpaceView,
			public:   map[string]bool{"space:root/child": true},
			wantKey:  "space:root/child",
			want:     true,
		},
		{
			name:     "private space",
			resource: types.Resource{Type: enum.ResourceTypeSpace, Identifier: "child"},
			scope:    types.Scope{SpacePath: "root"},
			perm:     enum.PermissionSpaceView,
			wantKey:  "space:root/child",
		},
		{
			name:     "space edit is never public",
			resource: types.Resource{Type: enum.ResourceTypeSpace, Identifier: "child"},
			scope:    types.Scope{SpacePath: "root"},
			perm:     enum.PermissionSpaceDelete,
		},
		{
			name:     "public repo view",
			resource: types.Resource{Type: enum.ResourceTypeRepo, Identifier: "repo"},
			scope:    types.Scope{SpacePath: "root"},
			perm:     enum.PermissionRepoView,
			public:   map[string]bool{"repository:root/repo": true},
			wantKey:  "repository:root/repo",
			want:     true,
		},
		{
			name:     "repo push is never public",
			resource: types.Resource{Type: enum.ResourceTypeRepo, Identifier: "repo"},
			scope:    types.Scope{SpacePath: "root"},
			perm:     enum.PermissionRepoPush,
		},
		{
			name:     "repo space-scope check uses the space",
			resource: types.Resource{Type: enum.ResourceTypeRepo},
			scope:    types.Scope{SpacePath: "root"},
			perm:     enum.PermissionSpaceView,
			public:   map[string]bool{"space:root": true},
			wantKey:  "space:root",
			want:     true,
		},
		{
			name:     "pipeline view follows repo visibility",
			resource: types.Resource{Type: enum.ResourceTypePipeline, Identifier: "pipe"},
			scope:    types.Scope{SpacePath: "root", Repo: "repo"},
			perm:     enum.PermissionPipelineView,
			public:   map[string]bool{"repository:root/repo": true},
			wantKey:  "repository:root/repo",
			want:     true,
		},
		{
			name:     "public registry view",
			resource: types.Resource{Type: enum.ResourceTypeRegistry, Identifier: "reg"},
			scope:    types.Scope{SpacePath: "root"},
			perm:     enum.PermissionArtifactsDownload,
			public:   map[string]bool{"registry:root/reg": true},
			wantKey:  "registry:root/reg",
			want:     true,
		},
		{
			name:     "registry space-scope check",
			resource: types.Resource{Type: enum.ResourceTypeRegistry},
			scope:    types.Scope{SpacePath: "root"},
			perm:     enum.PermissionSpaceView,
			wantKey:  "space:root",
		},
		{
			name:     "registry edit is never public",
			resource: types.Resource{Type: enum.ResourceTypeRegistry, Identifier: "reg"},
			scope:    types.Scope{SpacePath: "root"},
			perm:     enum.PermissionRepoPush,
		},
		{
			name:     "unsupported resource types are not public",
			resource: types.Resource{Type: enum.ResourceTypeSecret, Identifier: "s"},
			scope:    types.Scope{SpacePath: "root"},
			perm:     enum.PermissionRepoView,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pa := &infPublicAccess{public: tt.public}
			got, err := CheckPublicAccess(ctx, pa, &tt.scope, &tt.resource, tt.perm)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			if tt.wantKey == "" {
				require.Empty(t, pa.calls)
			} else {
				require.Equal(t, []string{tt.wantKey}, pa.calls)
			}
		})
	}

	pa := &infPublicAccess{err: errors.New("lookup failed")}
	_, err := CheckPublicAccess(ctx, pa, &types.Scope{SpacePath: "root"},
		&types.Resource{Type: enum.ResourceTypeSpace, Identifier: "c"}, enum.PermissionSpaceView)
	require.ErrorContains(t, err, "failed to check public accessabillity")
}

func TestInfRoleHasPermission(t *testing.T) {
	require.True(t, roleHasPermission(enum.MembershipRoleReader, enum.PermissionRepoView))
	require.False(t, roleHasPermission(enum.MembershipRoleReader, enum.PermissionRepoPush))
	require.True(t, roleHasPermission(enum.MembershipRoleContributor, enum.PermissionRepoPush))
	require.False(t, roleHasPermission("unknown-role", enum.PermissionRepoView))
}

func infPermissionGetter(spaces *infSpaces, ms *infMembershipStore) permissionCacheGetter {
	return permissionCacheGetter{spaceFinder: spaces.finder(), membershipStore: ms}
}

func TestInfPermissionCacheGetter_Find(t *testing.T) {
	ctx := context.Background()
	root := &types.SpaceCore{ID: 1, Path: "root"}
	child := &types.SpaceCore{ID: 2, ParentID: 1, Path: "root/child"}
	grand := &types.SpaceCore{ID: 3, ParentID: 2, Path: "root/child/grand"}
	spaces := infNewSpaces(root, child, grand)

	key := func(ref string, perm enum.Permission) PermissionCacheKey {
		return PermissionCacheKey{PrincipalID: 9, SpaceRef: ref, Permission: perm}
	}
	member := func(spaceID int64, role enum.MembershipRole) map[types.MembershipKey]*types.Membership {
		return map[types.MembershipKey]*types.Membership{
			{SpaceID: spaceID, PrincipalID: 9}: {Role: role},
		}
	}

	t.Run("direct membership", func(t *testing.T) {
		g := infPermissionGetter(spaces, &infMembershipStore{memberships: member(3, enum.MembershipRoleReader)})
		ok, err := g.Find(ctx, key("root/child/grand", enum.PermissionRepoView))
		require.NoError(t, err)
		require.True(t, ok)
	})
	t.Run("membership in direct role lacks the permission", func(t *testing.T) {
		g := infPermissionGetter(spaces, &infMembershipStore{memberships: member(3, enum.MembershipRoleReader)})
		ok, err := g.Find(ctx, key("root/child/grand", enum.PermissionRepoPush))
		require.NoError(t, err)
		require.False(t, ok)
	})
	t.Run("membership is inherited from parents", func(t *testing.T) {
		g := infPermissionGetter(spaces, &infMembershipStore{memberships: member(1, enum.MembershipRoleContributor)})
		ok, err := g.Find(ctx, key("root/child/grand", enum.PermissionRepoPush))
		require.NoError(t, err)
		require.True(t, ok)
	})
	t.Run("no membership at all", func(t *testing.T) {
		g := infPermissionGetter(spaces, &infMembershipStore{})
		ok, err := g.Find(ctx, key("root/child", enum.PermissionRepoView))
		require.NoError(t, err)
		require.False(t, ok)
	})
	t.Run("deleted space falls back to the first existing ancestor", func(t *testing.T) {
		g := infPermissionGetter(spaces, &infMembershipStore{memberships: member(2, enum.MembershipRoleReader)})
		ok, err := g.Find(ctx, key("root/child/missing", enum.PermissionRepoView))
		require.NoError(t, err)
		require.True(t, ok)
	})
	t.Run("nothing exists on the path", func(t *testing.T) {
		g := infPermissionGetter(spaces, &infMembershipStore{})
		ok, err := g.Find(ctx, key("other/space", enum.PermissionRepoView))
		require.NoError(t, err)
		require.False(t, ok)
	})
	t.Run("membership store failure", func(t *testing.T) {
		g := infPermissionGetter(spaces, &infMembershipStore{err: errors.New("db down")})
		_, err := g.Find(ctx, key("root", enum.PermissionRepoView))
		require.ErrorContains(t, err, "failed to find membership")
	})
	t.Run("space lookup failure", func(t *testing.T) {
		broken := infNewSpaces(root)
		broken.errID = errors.New("cache down")
		g := infPermissionGetter(broken, &infMembershipStore{})
		_, err := g.Find(ctx, key("root", enum.PermissionRepoView))
		require.ErrorContains(t, err, "failed to find an existing space")
	})
}

func infMembershipAuthorizer(
	spaces *infSpaces,
	ms *infMembershipStore,
	pa *infPublicAccess,
) *MembershipAuthorizer {
	finder := spaces.finder()
	cache := infMapCache[PermissionCacheKey, bool]{get: func(key PermissionCacheKey) (bool, error) {
		return infPermissionGetter(spaces, ms).Find(context.Background(), key)
	}}
	return NewMembershipAuthorizer(cache, finder, pa)
}

func TestInfMembershipAuthorizer_Check(t *testing.T) {
	ctx := context.Background()
	root := &types.SpaceCore{ID: 1, Path: "root"}
	child := &types.SpaceCore{ID: 2, ParentID: 1, Path: "root/child"}
	spaces := infNewSpaces(root, child)
	ms := &infMembershipStore{memberships: map[types.MembershipKey]*types.Membership{
		{SpaceID: 1, PrincipalID: 9}: {Role: enum.MembershipRoleReader},
	}}
	authorizer := infMembershipAuthorizer(spaces, ms, &infPublicAccess{public: map[string]bool{
		"repository:root/child/public-repo": true,
	}})

	userSession := func(md auth.Metadata) *auth.Session {
		return &auth.Session{Principal: types.Principal{ID: 9, UID: "alice", Type: enum.PrincipalTypeUser}, Metadata: md}
	}
	repoRes := func(id string) *types.Resource { return &types.Resource{Type: enum.ResourceTypeRepo, Identifier: id} }
	scope := &types.Scope{SpacePath: "root/child"}

	t.Run("public access wins", func(t *testing.T) {
		ok, err := authorizer.Check(ctx, userSession(&auth.EmptyMetadata{}), scope, repoRes("public-repo"),
			enum.PermissionRepoView)
		require.NoError(t, err)
		require.True(t, ok)
	})
	t.Run("public access errors are returned", func(t *testing.T) {
		a := infMembershipAuthorizer( //nolint:contextcheck // the helper builds the authorizer without a request context.
			spaces, ms, &infPublicAccess{err: errors.New("pa failed")})
		_, err := a.Check(ctx, userSession(nil), scope, repoRes("r"), enum.PermissionRepoView)
		require.ErrorContains(t, err, "failed to check public access")
	})
	t.Run("admins can do everything", func(t *testing.T) {
		admin := userSession(&auth.EmptyMetadata{})
		admin.Principal.Admin = true
		ok, err := authorizer.Check(ctx, admin, scope, repoRes("r"), enum.PermissionRepoPush)
		require.NoError(t, err)
		require.True(t, ok)
	})
	t.Run("membership permission cache", func(t *testing.T) {
		ok, err := authorizer.Check(ctx, userSession(&auth.EmptyMetadata{}), scope, repoRes("r"),
			enum.PermissionRepoView)
		require.NoError(t, err)
		require.True(t, ok)

		ok, err = authorizer.Check(ctx, userSession(&auth.EmptyMetadata{}), scope, repoRes("r"),
			enum.PermissionRepoPush)
		require.NoError(t, err)
		require.False(t, ok)
	})
	t.Run("space resources append the identifier to the scope", func(t *testing.T) {
		ok, err := authorizer.Check(ctx, userSession(nil), &types.Scope{SpacePath: "root"},
			&types.Resource{Type: enum.ResourceTypeSpace, Identifier: "child"}, enum.PermissionRepoView)
		require.NoError(t, err)
		require.True(t, ok)
	})
	t.Run("space scoped resource types use the scope path", func(t *testing.T) {
		for _, rt := range []enum.ResourceType{
			enum.ResourceTypeServiceAccount, enum.ResourceTypePipeline, enum.ResourceTypeSecret,
			enum.ResourceTypeConnector, enum.ResourceTypeTemplate, enum.ResourceTypeGitspace,
			enum.ResourceTypeInfraProvider, enum.ResourceTypeRegistry,
		} {
			ok, err := authorizer.Check(ctx, userSession(nil), scope,
				&types.Resource{Type: rt, Identifier: "x"}, enum.PermissionRepoView)
			require.NoError(t, err, rt)
			require.True(t, ok, rt)
		}
	})
	t.Run("user resources", func(t *testing.T) {
		self := &types.Resource{Type: enum.ResourceTypeUser, Identifier: "alice"}
		other := &types.Resource{Type: enum.ResourceTypeUser, Identifier: "bob"}

		ok, _ := authorizer.Check(ctx, userSession(nil), scope, self, enum.PermissionUserEdit)
		require.True(t, ok, "users may edit themselves")
		ok, _ = authorizer.Check(ctx, userSession(nil), scope, other, enum.PermissionUserEdit)
		require.False(t, ok)
		ok, _ = authorizer.Check(ctx, userSession(nil), scope, other, enum.PermissionUserView)
		require.True(t, ok, "everybody can view users")
		ok, _ = authorizer.Check(ctx, userSession(nil), scope, self, enum.PermissionUserEditAdmin)
		require.False(t, ok)
	})
	t.Run("services and unknown resources are denied", func(t *testing.T) {
		for _, rt := range []enum.ResourceType{enum.ResourceTypeService, "UNKNOWN"} {
			ok, err := authorizer.Check(ctx, userSession(nil), scope, &types.Resource{Type: rt}, enum.PermissionRepoView)
			require.NoError(t, err)
			require.False(t, ok, rt)
		}
	})
	t.Run("unknown metadata impacting authorization is rejected", func(t *testing.T) {
		_, err := authorizer.Check(ctx, userSession(&infImpactfulMetadata{}), scope, repoRes("r"),
			enum.PermissionRepoView)
		require.ErrorContains(t, err, "unknown metadata that impacts authorization")
	})
	t.Run("harmless metadata falls back to memberships", func(t *testing.T) {
		ok, err := authorizer.Check(ctx, userSession(&auth.TokenMetadata{}), scope, repoRes("r"),
			enum.PermissionRepoView)
		require.NoError(t, err)
		require.True(t, ok)
	})
}

type infImpactfulMetadata struct{}

func (*infImpactfulMetadata) ImpactsAuthorization() bool { return true }

func TestInfMembershipAuthorizer_EphemeralMembership(t *testing.T) {
	ctx := context.Background()
	root := &types.SpaceCore{ID: 1, Path: "root"}
	other := &types.SpaceCore{ID: 5, Path: "other"}
	spaces := infNewSpaces(root, other)
	authorizer := infMembershipAuthorizer(spaces, &infMembershipStore{}, &infPublicAccess{})

	session := func(spaceID int64, role enum.MembershipRole) *auth.Session {
		return &auth.Session{
			Principal: types.Principal{ID: 9},
			Metadata:  &auth.MembershipMetadata{SpaceID: spaceID, Role: role},
		}
	}
	res := &types.Resource{Type: enum.ResourceTypeRepo, Identifier: "r"}

	ok, err := authorizer.Check(ctx, session(1, enum.MembershipRoleContributor),
		&types.Scope{SpacePath: "root/child"}, res, enum.PermissionRepoPush)
	require.NoError(t, err)
	require.True(t, ok)

	_, err = authorizer.Check(ctx, session(1, enum.MembershipRoleReader),
		&types.Scope{SpacePath: "root/child"}, res, enum.PermissionRepoPush)
	require.ErrorContains(t, err, "outside of ephemeral membership role")

	_, err = authorizer.Check(ctx, session(5, enum.MembershipRoleContributor),
		&types.Scope{SpacePath: "root/child"}, res, enum.PermissionRepoPush)
	require.ErrorContains(t, err, "outside of ephemeral membership scope")

	_, err = authorizer.Check(ctx, session(404, enum.MembershipRoleContributor),
		&types.Scope{SpacePath: "root"}, res, enum.PermissionRepoPush)
	require.ErrorContains(t, err, "failed to find space")
}

func TestInfMembershipAuthorizer_AccessPermissions(t *testing.T) {
	ctx := context.Background()
	root := &types.SpaceCore{ID: 1, Path: "root"}
	spaces := infNewSpaces(root)
	authorizer := infMembershipAuthorizer(spaces, &infMembershipStore{}, &infPublicAccess{})
	res := &types.Resource{Type: enum.ResourceTypeRepo, Identifier: "r"}
	scope := &types.Scope{SpacePath: "root"}

	session := func(perms *jwt.SubClaimsAccessPermissions) *auth.Session {
		return &auth.Session{
			Principal: types.Principal{ID: 9},
			Metadata:  &auth.AccessPermissionMetadata{AccessPermissions: perms},
		}
	}

	granted := &jwt.SubClaimsAccessPermissions{Permissions: []jwt.AccessPermissions{
		{SpaceID: 7, Permissions: []enum.Permission{enum.PermissionRepoPush}},
		{SpaceID: 1, Permissions: []enum.Permission{enum.PermissionRepoView, enum.PermissionRepoPush}},
	}}
	ok, err := authorizer.Check(ctx, session(granted), scope, res, enum.PermissionRepoPush)
	require.NoError(t, err)
	require.True(t, ok)

	_, err = authorizer.Check(ctx, session(granted), scope, res, enum.PermissionSpaceDelete)
	require.ErrorContains(t, err, "no space_delete permission provided")

	_, err = authorizer.Check(ctx, session(&jwt.SubClaimsAccessPermissions{}), scope, res, enum.PermissionRepoView)
	require.ErrorContains(t, err, "no repo_view permission provided")

	_, err = authorizer.Check(ctx, session(granted), &types.Scope{SpacePath: "missing"}, res, enum.PermissionRepoView)
	require.ErrorContains(t, err, "failed to find space by ref")
}

func TestInfMembershipAuthorizer_CheckAllAndMany(t *testing.T) {
	ctx := context.Background()
	root := &types.SpaceCore{ID: 1, Path: "root"}
	spaces := infNewSpaces(root)
	ms := &infMembershipStore{memberships: map[types.MembershipKey]*types.Membership{
		{SpaceID: 1, PrincipalID: 9}: {Role: enum.MembershipRoleReader},
	}}
	authorizer := infMembershipAuthorizer(spaces, ms, &infPublicAccess{})
	session := &auth.Session{Principal: types.Principal{ID: 9}, Metadata: &auth.EmptyMetadata{}}

	allowed := types.PermissionCheck{
		Scope:      types.Scope{SpacePath: "root"},
		Resource:   types.Resource{Type: enum.ResourceTypeRepo, Identifier: "r"},
		Permission: enum.PermissionRepoView,
	}
	denied := allowed
	denied.Permission = enum.PermissionRepoPush

	ok, err := authorizer.CheckAll(ctx, session, allowed, allowed)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = authorizer.CheckAll(ctx, session, allowed, denied, allowed)
	require.NoError(t, err)
	require.False(t, ok)
	ok, err = authorizer.CheckAll(ctx, session)
	require.NoError(t, err)
	require.True(t, ok, "no checks means nothing is denied")

	results, err := authorizer.CheckMany(ctx, session, allowed, denied, allowed)
	require.NoError(t, err)
	require.Equal(t, []bool{true, false, true}, results)

	// errors abort both operations
	failing := allowed
	failing.Resource.Type = enum.ResourceTypeSecret
	ms.err = errors.New("db down")
	_, err = authorizer.CheckMany(ctx, session, failing)
	require.Error(t, err)
	ok, err = authorizer.CheckAll(ctx, session, failing)
	require.Error(t, err)
	require.False(t, ok)
}

func TestInfNewPermissionCache(t *testing.T) {
	root := &types.SpaceCore{ID: 1, Path: "root"}
	ms := &infMembershipStore{memberships: map[types.MembershipKey]*types.Membership{
		{SpaceID: 1, PrincipalID: 9}: {Role: enum.MembershipRoleReader},
	}}
	cache := NewPermissionCache(infNewSpaces(root).finder(), ms, time.Minute)

	ok, err := cache.Get(context.Background(), PermissionCacheKey{
		PrincipalID: 9, SpaceRef: "root", Permission: enum.PermissionRepoView,
	})
	require.NoError(t, err)
	require.True(t, ok)
}
