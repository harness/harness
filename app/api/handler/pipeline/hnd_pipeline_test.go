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

package pipeline

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	pipelinectrl "github.com/harness/gitness/app/api/controller/pipeline"
	"github.com/harness/gitness/app/api/request"
	pipelineevents "github.com/harness/gitness/app/events/pipeline"
	"github.com/harness/gitness/app/store/database"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func hndPipeServe(
	t *testing.T, b *hndBase, h http.HandlerFunc, body string, params map[string]string,
) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(body))
	routeCtx := chi.NewRouteContext()
	for k, v := range params {
		routeCtx.URLParams.Add(k, v)
	}
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, routeCtx)
	ctx = request.WithAuthSession(ctx, b.session())
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestHndPipelineHandlers_badBody(t *testing.T) {
	b := hndNewBase(t)
	pipelineStore := database.NewPipelineStore(b.db)
	triggerStore := database.NewTriggerStore(b.db)
	ctrl := pipelinectrl.NewController(
		hndAuthorizer{mode: hndAllow}, triggerStore, pipelineStore, pipelineevents.Reporter{}, b.repoFinder)
	params := map[string]string{request.PathParamRepoRef: "root/r1"}

	for _, h := range []http.HandlerFunc{HandleCreate(ctrl), HandleUpdate(ctrl)} {
		w := hndPipeServe(t, b, h, `{`, params)
		require.Equal(t, http.StatusBadRequest, w.Code)
	}
	w := hndPipeServe(t, b, HandleFind(ctrl), "", map[string]string{
		request.PathParamRepoRef:            "root/r1",
		request.PathParamPipelineIdentifier: "p1",
	})
	require.NotEqual(t, http.StatusOK, w.Code)
}
