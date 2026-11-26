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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	executionctrl "github.com/harness/gitness/app/api/controller/execution"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/store/database"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func hndExecServe(
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

func TestHndExecutionHandlers_badBody(t *testing.T) {
	b := hndNewBase(t)
	ctrl := executionctrl.NewController(
		b.tx, hndAuthorizer{mode: hndAllow}, database.NewExecutionStore(b.db), nil, nil, nil, nil,
		database.NewStageStore(b.db), database.NewPipelineStore(b.db), b.repoFinder,
	)
	params := map[string]string{request.PathParamRepoRef: "root/r1", request.PathParamPipelineIdentifier: "p1"}
	w := hndExecServe(t, b, HandleCreate(ctrl), `{`, params)
	require.NotEqual(t, http.StatusOK, w.Code)
}
