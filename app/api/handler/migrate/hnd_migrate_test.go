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

package migrate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/harness/gitness/app/api/controller/limiter"
	migratec "github.com/harness/gitness/app/api/controller/migrate"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

type hndRequest struct {
	method string
	params map[string]string
	body   string
}

func hndServe(t *testing.T, b *hndBase, h http.HandlerFunc, req hndRequest) *httptest.ResponseRecorder {
	t.Helper()
	method := req.method
	if method == "" {
		method = http.MethodPost
	}
	httpReq := httptest.NewRequestWithContext(t.Context(), method, "/", strings.NewReader(req.body))
	routeCtx := chi.NewRouteContext()
	for k, v := range req.params {
		routeCtx.URLParams.Add(k, v)
	}
	ctx := context.WithValue(httpReq.Context(), chi.RouteCtxKey, routeCtx)
	ctx = request.WithAuthSession(ctx, b.session())
	httpReq = httpReq.WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httpReq)
	return w
}

type hndEnv struct {
	*hndBase
	ctrl  *migratec.Controller
	space *types.Space
	repo  *types.Repository
}

func hndNewEnv(t *testing.T) *hndEnv {
	t.Helper()
	b := hndNewBase(t)
	space := b.createSpace(t, "root", 0)
	repo := b.createRepo(t, space, "repo1")
	ctrl := migratec.NewController(
		hndAuthorizer{mode: hndAllow}, nil, nil, nil, nil, nil, nil, nil,
		limiter.NewResourceLimiter(), audit.New(), check.RepoIdentifierDefault,
		b.tx, b.spaces, b.repos, b.spaceFinder, b.repoFinder, nil,
	)
	return &hndEnv{hndBase: b, ctrl: ctrl, space: space, repo: repo}
}

func TestHndMigrateHandlers_badBody(t *testing.T) {
	e := hndNewEnv(t)
	repoRef := "root/repo1"
	handlers := []struct {
		name string
		h    http.HandlerFunc
	}{
		{"create repo", HandleCreateRepo(e.ctrl)},
		{"update state", HandleUpdateRepoState(e.ctrl)},
		{"labels", HandleLabels(e.ctrl)},
		{"rules", HandleRules(e.ctrl)},
		{"pullreqs", HandlePullRequests(e.ctrl)},
		{"webhooks", HandleWebhooks(e.ctrl)},
	}
	for _, tc := range handlers {
		t.Run(tc.name, func(t *testing.T) {
			w := hndServe(t, e.hndBase, tc.h, hndRequest{
				params: map[string]string{request.PathParamRepoRef: repoRef},
				body:   `{`,
			})
			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestHndMigrateHandlers_missingRepoRef(t *testing.T) {
	e := hndNewEnv(t)
	w := hndServe(t, e.hndBase, HandleUpdateRepoState(e.ctrl), hndRequest{body: `{}`})
	require.NotEqual(t, http.StatusOK, w.Code)
}
