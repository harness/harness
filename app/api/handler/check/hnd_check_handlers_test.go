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

package pullreq

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	checkctrl "github.com/harness/gitness/app/api/controller/check"
	"github.com/harness/gitness/app/api/request"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func hndCheckServe(t *testing.T, h http.HandlerFunc, params map[string]string, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "/", strings.NewReader(body))
	routeCtx := chi.NewRouteContext()
	for k, v := range params {
		routeCtx.URLParams.Add(k, v)
	}
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, routeCtx))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestHndCheckHandlers_missingParams(t *testing.T) {
	ctrl := &checkctrl.Controller{}
	w := hndCheckServe(t, HandleCheckList(ctrl), nil, "")
	require.NotEqual(t, http.StatusOK, w.Code)

	w = hndCheckServe(t, HandleCheckReport(ctrl), map[string]string{
		request.PathParamRepoRef: "root/r1",
	}, "")
	require.NotEqual(t, http.StatusOK, w.Code)
}
