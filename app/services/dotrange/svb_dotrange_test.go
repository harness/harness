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

package dotrange

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/git"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

var errSvbBoom = errors.New("boom")

const (
	svbShaA = "1111111111111111111111111111111111111111"
	svbShaB = "2222222222222222222222222222222222222222"
)

func TestSvbParsePathAndString(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		want    DotRange
		wantStr string
		wantErr bool
	}{
		{
			name: "two dots", path: "main..feature",
			want:    DotRange{BaseRef: "main", HeadRef: "feature"},
			wantStr: "main..feature",
		},
		{
			name: "three dots use merge base", path: "main...feature",
			want:    DotRange{BaseRef: "main", HeadRef: "feature", MergeBase: true},
			wantStr: "main...feature",
		},
		{
			name: "upstream head", path: "main..upstream:feature",
			want:    DotRange{BaseRef: "main", HeadRef: "feature", HeadUpstream: true},
			wantStr: "main..upstream:feature",
		},
		{
			name: "upstream base with merge base", path: "upstream:main...feature",
			want:    DotRange{BaseRef: "main", HeadRef: "feature", BaseUpstream: true, MergeBase: true},
			wantStr: "upstream:main...feature",
		},
		{name: "no dots", path: "main", wantErr: true},
		{name: "both upstream", path: "upstream:a..upstream:b", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParsePath(tc.path)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
			require.Equal(t, tc.wantStr, got.String())
		})
	}
}

func TestSvbMake(t *testing.T) {
	got, err := Make("upstream:a", "b", true)
	require.NoError(t, err)
	require.True(t, got.BaseUpstream)
	require.False(t, got.HeadUpstream)
	require.Equal(t, "a", got.BaseRef)

	_, err = Make("upstream:a", "upstream:b", false)
	require.ErrorContains(t, err, "Only one upstream reference")
}

type svbAuthorizer struct {
	authz.Authorizer

	allow bool
	err   error
}

func (a svbAuthorizer) Check(
	context.Context, *auth.Session, *types.Scope, *types.Resource, enum.Permission,
) (bool, error) {
	return a.allow, a.err
}

type svbURLProvider struct{ url.Provider }

func (svbURLProvider) GetInternalAPIURL(context.Context) string { return "http://localhost:3000" }

type svbGit struct {
	git.Interface

	refSHA     sha.SHA
	refErr     error
	revSHA     sha.SHA
	revErr     error
	fetchErr   error
	fetched    []sha.SHA
	fetchFrom  string
	gotRefName string
	gotRefType gitenum.RefType
	gotRev     string
}

func (g *svbGit) GetRef(_ context.Context, p git.GetRefParams) (git.GetRefResponse, error) {
	g.gotRefName, g.gotRefType = p.Name, p.Type
	return git.GetRefResponse{SHA: g.refSHA}, g.refErr
}

func (g *svbGit) ResolveRevision(_ context.Context, p git.ResolveRevisionParams) (git.ResolveRevisionOutput, error) {
	g.gotRev = p.Revision
	return git.ResolveRevisionOutput{SHA: g.revSHA}, g.revErr
}

func (g *svbGit) FetchObjects(_ context.Context, p *git.FetchObjectsParams) (git.FetchObjectsOutput, error) {
	g.fetched, g.fetchFrom = p.ObjectSHAs, p.Source
	return git.FetchObjectsOutput{}, g.fetchErr
}

type svbRepoIDCache struct {
	store.RepoIDCache

	err error
}

func (f *svbRepoIDCache) Get(_ context.Context, id int64) (*types.RepositoryCore, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &types.RepositoryCore{ID: id, GitUID: "upstream-uid", Path: "acct/org/proj/upstream"}, nil
}

type svbEnv struct {
	svc   *Service
	git   *svbGit
	cache *svbRepoIDCache
}

func svbNewEnv(allow bool, authErr error) *svbEnv {
	env := &svbEnv{
		git:   &svbGit{refSHA: sha.Must(svbShaA), revSHA: sha.Must(svbShaB)},
		cache: &svbRepoIDCache{},
	}
	env.svc = &Service{
		git:         env.git,
		repoFinder:  refcache.NewRepoFinder(nil, nil, env.cache, nil, cache.Evictor[*types.RepositoryCore]{}),
		urlProvider: svbURLProvider{},
		authorizer:  svbAuthorizer{allow: allow, err: authErr},
	}
	return env
}

func svbSession() *auth.Session {
	return &auth.Session{Principal: types.Principal{ID: 5}, Metadata: &auth.EmptyMetadata{}}
}

func svbFork() *types.RepositoryCore {
	return &types.RepositoryCore{ID: 2, ForkID: 1, GitUID: "fork-uid", Path: "acct/org/proj/fork"}
}

func TestSvbFetchUpstreamBranch(t *testing.T) {
	ctx := context.Background()

	base := svbNewEnv(true, nil)
	got, upstream, err := base.svc.FetchUpstreamBranch(ctx, svbSession(), svbFork(), "main")
	require.NoError(t, err)
	require.Equal(t, svbShaA, got.String())
	require.Equal(t, int64(1), upstream.ID)
	require.Equal(t, "main", base.git.gotRefName)
	require.Equal(t, gitenum.RefTypeBranch, base.git.gotRefType)
	require.Equal(t, "upstream-uid", base.git.fetchFrom)
	require.Equal(t, []sha.SHA{sha.Must(svbShaA)}, base.git.fetched)

	t.Run("not a fork", func(t *testing.T) {
		_, _, err := base.svc.FetchUpstreamBranch(ctx, svbSession(), &types.RepositoryCore{ID: 2}, "main")
		require.ErrorContains(t, err, "not a fork")
	})

	t.Run("upstream lookup fails", func(t *testing.T) {
		env := svbNewEnv(true, nil)
		env.cache.err = errSvbBoom
		_, _, err := env.svc.FetchUpstreamBranch(ctx, svbSession(), svbFork(), "main")
		require.ErrorIs(t, err, errSvbBoom)
	})

	t.Run("no access to upstream", func(t *testing.T) {
		env := svbNewEnv(false, nil)
		_, _, err := env.svc.FetchUpstreamBranch(ctx, svbSession(), svbFork(), "main")
		require.ErrorContains(t, err, "Not enough permissions")
	})

	t.Run("authorizer error", func(t *testing.T) {
		env := svbNewEnv(false, errSvbBoom)
		_, _, err := env.svc.FetchUpstreamBranch(ctx, svbSession(), svbFork(), "main")
		require.ErrorIs(t, err, errSvbBoom)
	})

	t.Run("git failures", func(t *testing.T) {
		env := svbNewEnv(true, nil)
		env.git.refErr = errSvbBoom
		_, _, err := env.svc.FetchUpstreamBranch(ctx, svbSession(), svbFork(), "main")
		require.ErrorIs(t, err, errSvbBoom)

		env = svbNewEnv(true, nil)
		env.git.fetchErr = errSvbBoom
		_, _, err = env.svc.FetchUpstreamBranch(ctx, svbSession(), svbFork(), "main")
		require.ErrorIs(t, err, errSvbBoom)
	})
}

func TestSvbFetchDotRangeObjectsFromUpstream(t *testing.T) {
	ctx := context.Background()

	t.Run("resolves upstream refs to SHAs", func(t *testing.T) {
		env := svbNewEnv(true, nil)
		dr := &DotRange{BaseRef: "main", BaseUpstream: true, HeadRef: "feature", HeadUpstream: false}
		require.NoError(t, env.svc.FetchDotRangeObjectsFromUpstream(ctx, svbSession(), svbFork(), dr))
		require.False(t, dr.BaseUpstream)
		require.Equal(t, svbShaB, dr.BaseRef)
		require.Equal(t, "feature", dr.HeadRef)
		require.Equal(t, "main", env.git.gotRev)

		dr = &DotRange{BaseRef: "main", HeadRef: "feature", HeadUpstream: true}
		require.NoError(t, env.svc.FetchDotRangeObjectsFromUpstream(ctx, svbSession(), svbFork(), dr))
		require.False(t, dr.HeadUpstream)
		require.Equal(t, svbShaB, dr.HeadRef)
		require.Equal(t, "main", dr.BaseRef)
	})

	t.Run("local range is left alone", func(t *testing.T) {
		env := svbNewEnv(true, nil)
		dr := &DotRange{BaseRef: "a", HeadRef: "b"}
		require.NoError(t, env.svc.FetchDotRangeObjectsFromUpstream(ctx, svbSession(), svbFork(), dr))
		require.Equal(t, DotRange{BaseRef: "a", HeadRef: "b"}, *dr)
		require.Empty(t, env.git.fetched)
	})

	t.Run("failures", func(t *testing.T) {
		env := svbNewEnv(true, nil)
		env.git.revErr = errSvbBoom
		dr := &DotRange{BaseRef: "a", BaseUpstream: true, HeadRef: "b"}
		require.ErrorIs(t, env.svc.FetchDotRangeObjectsFromUpstream(ctx, svbSession(), svbFork(), dr), errSvbBoom)

		dr = &DotRange{BaseRef: "a", HeadRef: "b", HeadUpstream: true}
		require.ErrorIs(t, env.svc.FetchDotRangeObjectsFromUpstream(ctx, svbSession(), svbFork(), dr), errSvbBoom)
	})
}

func TestSvbFetchCommitDivergenceObjectsFromUpstream(t *testing.T) {
	ctx := context.Background()
	env := svbNewEnv(true, nil)

	// The divergence request's To/From map to the dot range's Base/Head.
	div := &git.CommitDivergenceRequest{To: "upstream:main", From: "feature"}
	require.NoError(t, env.svc.FetchCommitDivergenceObjectsFromUpstream(ctx, svbSession(), svbFork(), div))
	require.Equal(t, svbShaB, div.To)
	require.Equal(t, "feature", div.From)

	div = &git.CommitDivergenceRequest{To: "upstream:a", From: "upstream:b"}
	require.Error(t, env.svc.FetchCommitDivergenceObjectsFromUpstream(ctx, svbSession(), svbFork(), div))

	env.git.revErr = errSvbBoom
	div = &git.CommitDivergenceRequest{To: "upstream:main", From: "feature"}
	require.ErrorIs(t, env.svc.FetchCommitDivergenceObjectsFromUpstream(ctx, svbSession(), svbFork(), div), errSvbBoom)
}
