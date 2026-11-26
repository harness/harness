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

package trigger

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	triggerctrl "github.com/harness/gitness/app/api/controller/trigger"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/store/database"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

type hndTrigReq struct {
	method string
	params map[string]string
	body   string
}

func hndTrigServe(t *testing.T, b *hndBase, h http.HandlerFunc, req hndTrigReq) *httptest.ResponseRecorder {
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

func hndTrigEnv(t *testing.T) (*hndBase, *triggerctrl.Controller) {
	t.Helper()
	b := hndNewBase(t)
	triggerStore := database.NewTriggerStore(b.db)
	pipelineStore := database.NewPipelineStore(b.db)
	ctrl := triggerctrl.NewController(hndAuthorizer{mode: hndAllow}, triggerStore, pipelineStore, b.repoFinder)
	return b, ctrl
}

func TestHndTriggerHandlers_validation(t *testing.T) {
	b, ctrl := hndTrigEnv(t)
	params := map[string]string{
		request.PathParamRepoRef:            "root/repo1",
		request.PathParamPipelineIdentifier: "pipe1",
	}
	handlers := []struct {
		name string
		h    http.HandlerFunc
	}{
		{"create", HandleCreate(ctrl)},
		{"update", HandleUpdate(ctrl)},
		{"find", HandleFind(ctrl)},
		{"list", HandleList(ctrl)},
		{"delete", HandleDelete(ctrl)},
	}
	for _, tc := range handlers {
		t.Run(tc.name+" bad json", func(t *testing.T) {
			w := hndTrigServe(t, b, tc.h, hndTrigReq{params: params, body: `{"`})
			if tc.name == "list" || tc.name == "find" || tc.name == "delete" {
				// GET-style handlers may not decode a body; still exercise handler entry.
				require.NotEqual(t, http.StatusOK, w.Code)
				return
			}
			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
	t.Run("missing pipeline", func(t *testing.T) {
		w := hndTrigServe(t, b, HandleCreate(ctrl), hndTrigReq{
			params: map[string]string{request.PathParamRepoRef: "root/repo1"},
			body:   `{}`,
		})
		require.NotEqual(t, http.StatusCreated, w.Code)
	})
}
