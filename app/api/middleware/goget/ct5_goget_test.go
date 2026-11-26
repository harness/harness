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

package goget

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/services/publickey"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/app/url"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

const ct5BaseRepoPath = "root/space/repo"

var errCt5Boom = errors.New("boom")

type ct5Cache[K comparable, V any] struct {
	values map[K]V
	err    error
}

func (c *ct5Cache[K, V]) Stats() (int64, int64) { return 0, 0 }

func (c *ct5Cache[K, V]) Get(_ context.Context, key K) (V, error) {
	var zero V
	if c.err != nil {
		return zero, c.err
	}
	v, ok := c.values[key]
	if !ok {
		return zero, gitnessstore.ErrResourceNotFound
	}
	return v, nil
}

func (c *ct5Cache[K, V]) Evict(context.Context, K) {}

type ct5RepoStore struct {
	store.RepoStore
}

func (ct5RepoStore) Find(_ context.Context, id int64) (*types.Repository, error) {
	return &types.Repository{ID: id, Path: ct5BaseRepoPath, DefaultBranch: "trunk"}, nil
}

type ct5Authorizer struct {
	authz.Authorizer
	allowed bool
}

func (a ct5Authorizer) Check(
	context.Context, *auth.Session, *types.Scope, *types.Resource, enum.Permission,
) (bool, error) {
	return a.allowed, nil
}

type ct5Public struct {
	publicaccess.Service
}

func (ct5Public) Get(context.Context, enum.PublicResourceType, string) (bool, error) {
	return false, nil
}

type ct5Favorites struct {
	store.FavoriteStore
}

func (ct5Favorites) Map(context.Context, int64, enum.ResourceType, []int64) (map[int64]bool, error) {
	return map[int64]bool{}, nil
}

type ct5URLs struct {
	url.Provider
	uiBase string
}

func (u ct5URLs) GenerateUIRepoURL(_ context.Context, repoPath string) string {
	return u.uiBase + "/" + repoPath
}

func (ct5URLs) GenerateGITCloneURL(_ context.Context, repoPath string) string {
	return "https://git.example.com/" + repoPath + ".git"
}

func (ct5URLs) GenerateGITCloneSSHURL(_ context.Context, repoPath string) string {
	return "ssh://git.example.com/" + repoPath + ".git"
}

func ct5NewRepoController(allowed bool, repoIDErr error, provider url.Provider) *repo.Controller {
	finder := refcache.NewRepoFinder(
		ct5RepoStore{},
		&ct5Cache[string, *types.SpacePath]{values: map[string]*types.SpacePath{
			"root/space": {Value: "root/space", SpaceID: 10},
		}},
		&ct5Cache[int64, *types.RepositoryCore]{
			values: map[int64]*types.RepositoryCore{
				1: {ID: 1, Identifier: "repo", Path: ct5BaseRepoPath},
			},
			err: repoIDErr,
		},
		&ct5Cache[types.RepoCacheKey, int64]{values: map[types.RepoCacheKey]int64{
			{SpaceID: 10, RepoIdentifier: "repo"}: 1,
		}},
		cache.Evictor[*types.RepositoryCore]{},
	)
	return repo.NewController(
		&types.Config{}, nil, provider, ct5Authorizer{allowed: allowed}, ct5RepoStore{},
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		refcache.SpaceFinder{}, finder,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		ct5Public{},
		nil, nil, nil, nil, nil, nil, nil,
		ct5Favorites{},
		publickey.SignatureVerifyService{}, nil, nil, nil, nil, nil, nil,
	)
}

func ct5Serve(
	t *testing.T, ctrl *repo.Controller, provider url.Provider, maxDepth int, importPath string,
) *httptest.ResponseRecorder {
	t.Helper()

	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("next must not run for go-get") })
	handler := Middleware(maxDepth, ctrl, provider)(next)

	req := httptest.NewRequest(http.MethodGet, "/x?go-get=1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(request.PathParamRepoRef, importPath)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = request.WithAuthSession(ctx, &auth.Session{Principal: types.Principal{ID: 7, UID: "alice"}})
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestCt5GogetMiddleware(t *testing.T) {
	https := ct5URLs{uiBase: "https://ui.example.com"}

	tests := []struct {
		name       string
		allowed    bool
		repoIDErr  error
		provider   url.Provider
		maxDepth   int
		importPath string
		wantStatus int
		wantBody   []string
		notBody    []string
	}{
		{
			name: "repo found", allowed: true, provider: https, maxDepth: 3, importPath: ct5BaseRepoPath,
			wantStatus: http.StatusOK,
			wantBody: []string{
				"go-import", "git.example.com/root/space/repo git https://git.example.com/root/space/repo.git",
				"https://ui.example.com/root/space/repo/files/trunk/~", "go get  git.example.com/root/space/repo",
			},
			notBody: []string{"--insecure"},
		},
		{
			name: "insecure ui url", allowed: true, provider: ct5URLs{uiBase: "http://ui.example.com"},
			maxDepth: 3, importPath: ct5BaseRepoPath,
			wantStatus: http.StatusOK, wantBody: []string{"--insecure"},
		},
		{
			name: "submodule suffix resolves to repo root", allowed: true, provider: https, maxDepth: 4,
			importPath: ct5BaseRepoPath + "/v2",
			wantStatus: http.StatusOK, wantBody: []string{"git.example.com/root/space/repo git"},
		},
		{
			name: "path truncated to max depth", allowed: true, provider: https, maxDepth: 3,
			importPath: ct5BaseRepoPath + "/v2/more",
			wantStatus: http.StatusOK, wantBody: []string{"git.example.com/root/space/repo git"},
		},
		{
			name: "unknown repo", allowed: true, provider: https, maxDepth: 3, importPath: "root/space/other",
			wantStatus: http.StatusNotFound,
		},
		{
			name: "single segment never matches", allowed: true, provider: https, maxDepth: 3, importPath: "root",
			wantStatus: http.StatusNotFound,
		},
		{
			name: "forbidden treated as missing", allowed: false, provider: https, maxDepth: 3,
			importPath: ct5BaseRepoPath, wantStatus: http.StatusNotFound,
		},
		{
			name: "unexpected error is reported", allowed: true, repoIDErr: errCt5Boom, provider: https,
			maxDepth: 3, importPath: ct5BaseRepoPath, wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := ct5NewRepoController(tt.allowed, tt.repoIDErr, tt.provider)
			rec := ct5Serve(t, ctrl, tt.provider, tt.maxDepth, tt.importPath)
			require.Equal(t, tt.wantStatus, rec.Code)
			for _, s := range tt.wantBody {
				require.Contains(t, rec.Body.String(), s)
			}
			for _, s := range tt.notBody {
				require.NotContains(t, rec.Body.String(), s)
			}
		})
	}
}
